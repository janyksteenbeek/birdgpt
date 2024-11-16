package processor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/janyksteenbeek/birdgpt/config"
	"github.com/janyksteenbeek/birdgpt/internal/gmail"
	"github.com/janyksteenbeek/birdgpt/internal/moneybird"
	"github.com/janyksteenbeek/birdgpt/internal/openai"
)

type Processor struct {
	cfg        *config.Config
	gmail      *gmail.Client
	moneybird  *moneybird.Client
	openai     *openai.Client
	lastUpdate time.Time
}

func New(cfg *config.Config, gmailClient *gmail.Client, moneybirdClient *moneybird.Client, openaiClient *openai.Client) *Processor {
	lastUpdate, err := time.Parse(time.RFC3339, cfg.App.LastUpdate)
	if err != nil {
		log.Fatal("Error parsing last update time: %v", err)
	}

	return &Processor{
		cfg:        cfg,
		gmail:      gmailClient,
		moneybird:  moneybirdClient,
		openai:     openaiClient,
		lastUpdate: lastUpdate,
	}
}

func (p *Processor) Run(ctx context.Context) error {
	// Process immediately on first run
	if err := p.processNewEmails(ctx); err != nil {
		log.Printf("Error processing emails: %v", err)
	}

	ticker := time.NewTicker(p.cfg.App.SleepTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := p.processNewEmails(ctx); err != nil {
				log.Printf("Error processing emails: %v", err)
			}

			p.lastUpdate = time.Now()
			if err := config.SaveConfig(p.cfg); err != nil {
				log.Printf("Error saving config: %v", err)
			}
		}
	}
}

func (p *Processor) processNewEmails(ctx context.Context) error {
	log.Printf("Checking for new emails since %v...", p.lastUpdate.Format(time.RFC3339))
	emails, err := p.gmail.FetchEmails(ctx, p.cfg.Gmail.SearchLabel, p.lastUpdate)
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	if len(emails) == 0 {
		log.Println("No new emails found")
		return nil
	}

	log.Printf("Found %d new emails", len(emails))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)
	errChan := make(chan error, len(emails))

	for _, email := range emails {
		wg.Add(1)
		go func(e gmail.Email) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			log.Printf("Started processing email: %s - %s", e.Subject, e.From)
			if err := p.processEmail(ctx, e); err != nil {
				log.Printf("Failed to process email %s: %v", e.Subject, err)
				errChan <- fmt.Errorf("failed to process email %s: %w", e.ID, err)
			}
		}(email)
	}
	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors while processing emails", len(errors))
	}

	return nil
}

func (p *Processor) processEmail(ctx context.Context, email gmail.Email) error {
	invoiceData, err := p.openai.ProcessInvoice(ctx, email.Body, email.Attachments)
	if err != nil {
		return fmt.Errorf("failed to process with OpenAI: %w", err)
	}

	if !invoiceData.IsInvoice {
		log.Printf("Email is not an invoice: %s", email.Subject)
		return nil
	}

	log.Printf("Invoice detected: %s - %s - €%.2f", invoiceData.CompanyName, invoiceData.InvoiceNumber, invoiceData.TotalAmount)

	valid, reason, err := p.openai.ValidateInvoice(ctx, invoiceData)
	if err != nil {
		return fmt.Errorf("failed to validate invoice: %w", err)
	}
	if !valid {
		return fmt.Errorf("invalid invoice data: %s", reason)
	}

	contacts, err := p.moneybird.SearchContacts(invoiceData.CompanyName)
	if err != nil {
		return fmt.Errorf("failed to search contacts: %w", err)
	}

	var contact *moneybird.Contact

	if len(contacts) == 0 {
		log.Printf("Creating new contact: %s", invoiceData.CompanyName)
		newContact := &moneybird.Contact{
			CompanyName: invoiceData.CompanyName,
			Email:       invoiceData.ContactInfo.Email,
			CustomerId:  invoiceData.KvkNumber,
			TaxNumber:   invoiceData.VatNumber,
			Address:     invoiceData.ContactInfo.Street,
			City:        invoiceData.ContactInfo.City,
			Country:     invoiceData.ContactInfo.Country,
			ZipCode:     invoiceData.ContactInfo.ZipCode,
		}
		contact, err = p.moneybird.CreateContact(newContact)
		if err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}
	} else {
		contact = &contacts[0]
		log.Printf("Using existing contact: %s", contact.CompanyName)
	}

	shouldShiftVAT := moneybird.IsEUCountry(contact.Country) &&
		contact.Country != p.cfg.Moneybird.Country &&
		contact.TaxNumber != ""

	if shouldShiftVAT {
		log.Printf("VAT will be shifted (EU B2B transaction with %s)", contact.Country)
	}

	invoice := &moneybird.PurchaseInvoice{
		ContactID: contact.ID,
		Reference: invoiceData.InvoiceNumber,
		Date:      invoiceData.InvoiceDate,
		DueDate:   invoiceData.DueDate,
		Details:   make([]moneybird.InvoiceDetail, len(invoiceData.Items)),
	}

	for i, item := range invoiceData.Items {
		taxRateID := p.moneybird.GetTaxRateID(item.TaxRate)
		if shouldShiftVAT {
			taxRateID = p.moneybird.GetVATShiftedTaxRateID()
		}

		invoice.Details[i] = moneybird.InvoiceDetail{
			Description: item.Description,
			Price:       item.Amount,
			TaxRateID:   taxRateID,
		}
	}

	_, err = p.moneybird.CreatePurchaseInvoice(invoice)
	if err != nil {
		return fmt.Errorf("failed to create purchase invoice: %w", err)
	}

	log.Printf("Successfully created purchase invoice for %s (€%.2f)", invoiceData.CompanyName, invoiceData.TotalAmount)
	return nil
}
