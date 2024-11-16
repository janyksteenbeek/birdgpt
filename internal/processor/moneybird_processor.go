package processor

import (
	"context"
	"fmt"
	"log"

	"github.com/janyksteenbeek/birdgpt/config"
	"github.com/janyksteenbeek/birdgpt/internal/moneybird"
	"github.com/janyksteenbeek/birdgpt/internal/openai"
)

type MoneybirdProcessor struct {
	cfg       *config.Config
	moneybird *moneybird.Client
}

func NewMoneybirdProcessor(cfg *config.Config, moneybirdClient *moneybird.Client) *MoneybirdProcessor {
	return &MoneybirdProcessor{
		cfg:       cfg,
		moneybird: moneybirdClient,
	}
}

func (p *MoneybirdProcessor) ProcessInvoice(ctx context.Context, invoiceData *openai.InvoiceData) error {
	contacts, err := p.moneybird.SearchContacts(invoiceData.CompanyName)
	if err != nil {
		return fmt.Errorf("failed to search contacts: %w", err)
	}

	var contact *moneybird.Contact

	if len(contacts) == 0 {
		log.Printf("Creating new contact: %s", invoiceData.CompanyName)
		contact, err = p.createContact(invoiceData)
		if err != nil {
			return err
		}
	} else {
		contact = &contacts[0]
		log.Printf("Using existing contact: %s", contact.CompanyName)
	}

	shouldShiftVAT := p.shouldShiftVAT(contact)
	if shouldShiftVAT {
		log.Printf("VAT will be shifted (EU B2B transaction with %s)", contact.Country)
	}

	invoice := p.createPurchaseInvoice(invoiceData, contact, shouldShiftVAT)
	if _, err = p.moneybird.CreatePurchaseInvoice(invoice); err != nil {
		return fmt.Errorf("failed to create purchase invoice: %w", err)
	}

	log.Printf("Successfully created purchase invoice for %s (€%.2f)", invoiceData.CompanyName, invoiceData.TotalAmount)
	return nil
}

func (p *MoneybirdProcessor) createContact(data *openai.InvoiceData) (*moneybird.Contact, error) {
	contact := &moneybird.Contact{
		CompanyName: data.CompanyName,
		Email:       data.ContactInfo.Email,
		CustomerId:  data.KvkNumber,
		TaxNumber:   data.VatNumber,
		Address:     data.ContactInfo.Street,
		City:        data.ContactInfo.City,
		Country:     data.ContactInfo.Country,
		ZipCode:     data.ContactInfo.ZipCode,
	}

	created, err := p.moneybird.CreateContact(contact)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	return created, nil
}

func (p *MoneybirdProcessor) shouldShiftVAT(contact *moneybird.Contact) bool {
	return moneybird.IsEUCountry(contact.Country) &&
		contact.Country != p.cfg.Moneybird.Country &&
		contact.TaxNumber != ""
}

func (p *MoneybirdProcessor) createPurchaseInvoice(data *openai.InvoiceData, contact *moneybird.Contact, shouldShiftVAT bool) *moneybird.PurchaseInvoice {
	invoice := &moneybird.PurchaseInvoice{
		ContactID:  contact.ID,
		Reference:  data.InvoiceNumber,
		Date:       data.InvoiceDate,
		DueDate:    data.DueDate,
		Details:    make([]moneybird.InvoiceDetail, len(data.Items)),
	}

	for i, item := range data.Items {
		taxRateID := p.moneybird.GetTaxRateID(item.TaxRate)
		if shouldShiftVAT {
			taxRateID = p.moneybird.GetVATShiftedTaxRateID()
		}

		invoice.Details[i] = moneybird.InvoiceDetail{
			Description: item.Description,
			Price:      item.Amount,
			TaxRateID:  taxRateID,
		}
	}

	return invoice
} 