package moneybird

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	adminID    string
	taxRates   map[float64]string
}

type TaxRate struct {
	ID          string  `json:"id"`
	Percentage  float64 `json:"percentage,string"`
	Name        string  `json:"name"`
	TaxRateType string  `json:"tax_rate_type"`
	Active      bool    `json:"active"`
}

func NewClient(token, adminID string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: time.Second * 30},
		baseURL:    "https://moneybird.com/api/v2",
		token:      token,
		adminID:    adminID,
		taxRates:   make(map[float64]string),
	}

	if err := c.initializeTaxRates(); err != nil {
		return nil, fmt.Errorf("initializing tax rates: %w", err)
	}

	return c, nil
}

func (c *Client) initializeTaxRates() error {
	resp, err := c.doRequest("GET", "tax_rates.json?filter=tax_rate_type:purchase_invoice,active:true", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rates []TaxRate
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return err
	}

	for _, rate := range rates {
		c.taxRates[rate.Percentage] = rate.ID
	}

	return nil
}

func (c *Client) GetTaxRateID(percentage float64) string {
	if id, ok := c.taxRates[percentage]; ok {
		return id
	}

	var closest float64
	var closestID string
	for rate, id := range c.taxRates {
		if closestID == "" || abs(percentage-rate) < abs(percentage-closest) {
			closest = rate
			closestID = id
		}
	}

	return closestID
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s/%s/%s", c.baseURL, c.adminID, path)
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (c *Client) GetVATShiftedTaxRateID() string {
	// Look for tax rate with 0% for intracommunity transactions
	return c.GetTaxRateID(0)
}
