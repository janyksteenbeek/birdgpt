package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"log"
)

type Client struct {
	client *openai.Client
}

type InvoiceData struct {
	IsInvoice     bool          `json:"is_invoice"`
	CompanyName   string        `json:"company_name,omitempty"`
	InvoiceNumber string        `json:"invoice_number,omitempty"`
	InvoiceDate   string        `json:"invoice_date,omitempty"`
	DueDate       string        `json:"due_date,omitempty"`
	Items         []InvoiceItem `json:"items,omitempty"`
	TotalAmount   float64       `json:"total_amount,omitempty"`
	TaxAmount     float64       `json:"tax_amount,omitempty"`
	ContactInfo   ContactInfo   `json:"contact_info,omitempty"`
	KvkNumber     string        `json:"kvk_number,omitempty"`
	VatNumber     string        `json:"vat_number,omitempty"`
}

type InvoiceItem struct {
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	TaxRate     float64 `json:"tax_rate"`
}

type ContactInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Street  string `json:"street"`
	City    string `json:"city"`
	ZipCode string `json:"zipcode"`
	Country string `json:"country"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		client: openai.NewClient(apiKey),
	}
}

func (c *Client) ProcessInvoice(ctx context.Context, emailBody string, attachments [][]byte) (*InvoiceData, error) {
	schema, err := jsonschema.GenerateSchemaForType(InvoiceData{})
	if err != nil {
		log.Fatalf("GenerateSchemaForType error: %v", err)
	}

	systemMsg := `You are an invoice processing assistant. First determine if the content contains an invoice. 
If it does, extract the relevant information. Pay special attention to KVK (Chamber of Commerce) and BTW (VAT) numbers, 
which are often found in the header or footer of Dutch invoices. BTW numbers typically start with NL and KVK numbers 
are 8 digits. Parse the address into separate components.

Only include the additional fields if is_invoice is true.
Consider invoice indicators like: payment terms, invoice numbers, line items, tax amounts.
For Dutch companies, always try to find the KVK and BTW numbers.
Always try to parse the full address into separate components.
Use ISO country codes for the country field.`

	var userContent string
	userContent += "Email content:\n" + emailBody + "\n\n"

	for i, attachment := range attachments {
		userContent += fmt.Sprintf("Attachment %d content:\n%s\n\n", i+1, base64.StdEncoding.EncodeToString(attachment))
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemMsg,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userContent,
				},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
					Name:   "invoice_schema",
					Schema: schema,
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	var invoiceData InvoiceData
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &invoiceData); err != nil {
		return nil, fmt.Errorf("parsing OpenAI response: %w", err)
	}

	if !invoiceData.IsInvoice {
		return &InvoiceData{IsInvoice: false}, nil
	}

	return &invoiceData, nil
}

func (c *Client) ValidateInvoice(ctx context.Context, invoice *InvoiceData) (bool, string, error) {
	if !invoice.IsInvoice {
		return false, "not an invoice", nil
	}

	prompt := fmt.Sprintf(`Validate this invoice data and respond with "VALID" or "INVALID" followed by a reason if invalid:

Company: %s
Invoice Number: %s
Total Amount: %.2f
Number of Items: %d
KVK Number: %s
VAT Number: %s
Address: %s, %s %s, %s

Validation rules:
1. All required fields must be present
2. KVK number should be 8 digits (if provided)
3. VAT number should start with NL and be in correct format (if provided)
4. Total amount should match line items
5. Address should have all components (street, city, zipcode, country)
6. Country code should be valid ISO code
`, invoice.CompanyName, invoice.InvoiceNumber, invoice.TotalAmount, len(invoice.Items),
		invoice.KvkNumber, invoice.VatNumber, invoice.ContactInfo.Street,
		invoice.ContactInfo.City, invoice.ContactInfo.ZipCode, invoice.ContactInfo.Country)

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return false, "", fmt.Errorf("validation failed: %w", err)
	}

	response := resp.Choices[0].Message.Content
	isValid := response[:5] == "VALID"
	reason := response[5:]
	if !isValid {
		reason = response[7:]
	}

	return isValid, reason, nil
}
