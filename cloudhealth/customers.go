package cloudhealth

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type Customers struct {
	Customers []*Customer `json:"customers"`
}

type Customer struct {
	ID                          int64                       `json:"id,omitempty"`
	Name                        string                      `json:"name"`
	Classification              string                      `json:"classification"`
	Address                     Address                     `json:"address"`
	PartnerBillingConfiguration PartnerBillingConfiguration `json:"partner_billing_configuration"`
	GeneratedExternalID         string                      `json:"generated_external_id"`
	Tags                        []Tag                       `json:"tags"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Address struct {
	Street1 string `json:"street1"`
	Street2 string `json:"street2"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zipcode"`
	Country string `json:"country"`
}

type PartnerBillingConfiguration struct {
	Enabled bool `json:"enabled"`
}

// GetTagValue returns a pointer to the tag value of the provided key.
// nil return value means that the key does not exist in the tags array.
func (c *Customer) GetTagValue(key string) *string {
	var value *string

	for _, t := range c.Tags {
		if t.Key == key {
			value = &t.Value
			break
		}
	}

	return value
}

func ListCustomers(page int64, m map[int64]*Customer) error {
	path := "/v1/customers"
	params := make(map[string][]string)
	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := Client.Get(path, params)
	if err != nil {
		return err
	}

	var response Customers
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.Customers) > 0 {
		for _, customer := range response.Customers {
			m[customer.ID] = customer
		}

		return ListCustomers(page+1, m)
	}

	return nil
}

func GetCustomer(id int64) (*Customer, error) {
	path := fmt.Sprintf("/v1/customers/%d", id)

	body, err := Client.Get(path, nil)
	if err != nil {
		return nil, err
	}

	var response Customer
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return &response, nil
}
