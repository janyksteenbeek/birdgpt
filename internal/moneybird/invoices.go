package moneybird

import (
	"encoding/json"
	"fmt"
	"io"
)

type PurchaseInvoice struct {
	ID        string          `json:"id"`
	ContactID string          `json:"contact_id"`
	Reference string          `json:"reference"`
	Date      string          `json:"date"`
	DueDate   string          `json:"due_date"`
	Details   []InvoiceDetail `json:"details_attributes"`
}

type InvoiceDetail struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	TaxRateID   string  `json:"tax_rate_id"`
}

func (c *Client) CreatePurchaseInvoice(invoice *PurchaseInvoice) (*PurchaseInvoice, error) {
	resp, err := c.doRequest("POST", "documents/purchase_invoices.json", map[string]interface{}{
		"purchase_invoice": invoice,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		// get body content
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var created PurchaseInvoice
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, err
	}

	return &created, nil
}
