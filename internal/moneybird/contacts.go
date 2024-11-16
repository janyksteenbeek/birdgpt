package moneybird

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Contact struct {
	ID          string `json:"id"`
	CompanyName string `json:"company_name"`
	FirstName   string `json:"firstname"`
	LastName    string `json:"lastname"`
	Email       string `json:"email"`
	CustomerId  string `json:"customer_id"`
	TaxNumber   string `json:"tax_number"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Country     string `json:"country"`
	ZipCode     string `json:"zipcode"`
}

func (c *Client) SearchContacts(query string) ([]Contact, error) {
	query = url.QueryEscape(query)
	resp, err := c.doRequest("GET", fmt.Sprintf("contacts.json?query=%s", query), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.Body)
	}

	var contacts []Contact
	if err := json.NewDecoder(resp.Body).Decode(&contacts); err != nil {
		return nil, err
	}

	return contacts, nil
}

func (c *Client) CreateContact(contact *Contact) (*Contact, error) {
	resp, err := c.doRequest("POST", "contacts.json", map[string]interface{}{
		"contact": contact,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var created Contact
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, err
	}

	return &created, nil
}

func IsEUCountry(countryCode string) bool {
	euCountries := map[string]bool{
		"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
		"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
		"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
		"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
		"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
		"ES": true, "SE": true,
	}
	return euCountries[countryCode]
}
