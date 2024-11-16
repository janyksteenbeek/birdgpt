package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ledongthuc/pdf"
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
		if isPDF(attachment) {
			text, err := extractTextFromPDF(attachment)
			if err != nil {
				return nil, fmt.Errorf("failed to extract text from PDF attachment %d: %w", i+1, err)
			}
			userContent += fmt.Sprintf("Attachment %d content (PDF):\n%s\n\n", i+1, text)
		} else {
			userContent += fmt.Sprintf("Attachment %d content:\n%s\n\n", i+1, base64.StdEncoding.EncodeToString(attachment))
		}
	}

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
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

func isPDF(data []byte) bool {
	return len(data) > 4 && string(data[:4]) == "%PDF"
}

func extractTextFromPDF(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var text string
	numPages := reader.NumPage()

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := reader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		extractedText, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", pageNum, err)
		}
		text += extractedText
	}

	return text, nil
}
