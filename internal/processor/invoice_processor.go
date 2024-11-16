package processor

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/janyksteenbeek/birdgpt/internal/gmail"
	"github.com/janyksteenbeek/birdgpt/internal/openai"
)

type InvoiceProcessor struct {
	openai *openai.Client
}

func NewInvoiceProcessor(openaiClient *openai.Client) *InvoiceProcessor {
	return &InvoiceProcessor{
		openai: openaiClient,
	}
}

func (p *InvoiceProcessor) ProcessEmail(ctx context.Context, email gmail.Email) (*openai.InvoiceData, error) {
	log.Printf("Processing email: %s - %s", email.Subject, email.From)
	
	invoiceData, err := p.openai.ProcessInvoice(ctx, email.Body, email.Attachments)
	if err != nil {
		return nil, fmt.Errorf("failed to process with OpenAI: %w", err)
	}

	if !invoiceData.IsInvoice {
		log.Printf("Email is not an invoice: %s", email.Subject)
		return nil, nil
	}

	log.Printf("Invoice detected: %s - %s - €%.2f", invoiceData.CompanyName, invoiceData.InvoiceNumber, invoiceData.TotalAmount)

	if err := p.validateInvoiceData(invoiceData); err != nil {
		return nil, fmt.Errorf("invoice validation failed: %w", err)
	}

	return invoiceData, nil
}

func (p *InvoiceProcessor) validateInvoiceData(invoice *openai.InvoiceData) error {
	if !invoice.IsInvoice {
		return fmt.Errorf("not an invoice")
	}

	if invoice.CompanyName == "" {
		return fmt.Errorf("company name is required")
	}

	if invoice.InvoiceNumber == "" {
		return fmt.Errorf("invoice number is required")
	}

	if invoice.TotalAmount <= 0 {
		return fmt.Errorf("invalid total amount: %.2f", invoice.TotalAmount)
	}

	if len(invoice.Items) == 0 {
		return fmt.Errorf("at least one invoice item is required")
	}

	// Validate KVK number if provided
	if invoice.KvkNumber != "" {
		kvkRegex := regexp.MustCompile(`^\d{8}$`)
		if !kvkRegex.MatchString(invoice.KvkNumber) {
			return fmt.Errorf("invalid KVK number format: %s", invoice.KvkNumber)
		}
	}

	// Validate country code
	if invoice.ContactInfo.Country != "" {
		if len(invoice.ContactInfo.Country) != 2 {
			return fmt.Errorf("invalid country code: %s", invoice.ContactInfo.Country)
		}
		invoice.ContactInfo.Country = strings.ToUpper(invoice.ContactInfo.Country)
	}

	// Validate total amount matches sum of items
	var total float64
	for _, item := range invoice.Items {
		if item.Amount <= 0 {
			return fmt.Errorf("invalid item amount: %.2f", item.Amount)
		}
		if item.TaxRate < 0 {
			return fmt.Errorf("invalid tax rate: %.2f", item.TaxRate)
		}
		total += item.Amount
	}

	if total != invoice.TotalAmount {
		return fmt.Errorf("total amount (%.2f) does not match sum of items (%.2f)", invoice.TotalAmount, total)
	}

	return nil
} 