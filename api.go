package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type BambooClient struct {
	subdomain string
	apikey    string
}

func NewBambooClient(subdomain, apikey string) *BambooClient {
	return &BambooClient{
		subdomain: subdomain,
		apikey:    apikey,
	}
}

func (c *BambooClient) request(ctx context.Context, path string, dst interface{}) error {
	url := fmt.Sprintf("https://api.bamboohr.com/api/gateway.php/%s/v1/%s", c.subdomain, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.apikey, "x")
	req.Header.Add("Accept", "application/json")
	req = req.WithContext(ctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: %s: %s", url, resp.Status, string(b))
	}
	return json.Unmarshal(b, dst)
}

func (c *BambooClient) EmployeeDirectory(ctx context.Context) (EmployeeDirectory, error) {
	var ret EmployeeDirectory
	err := c.request(ctx, "employees/directory", &ret)
	return ret, err
}

type EmployeeDirectory struct {
	Employees []struct {
		Department    string `json:"department"`
		DisplayName   string `json:"displayName"`
		Division      string `json:"division"`
		FirstName     string `json:"firstName"`
		Gender        string `json:"gender"`
		ID            string `json:"id"`
		JobTitle      string `json:"jobTitle"`
		LastName      string `json:"lastName"`
		Location      string `json:"location"`
		PhotoUploaded bool   `json:"photoUploaded"`
		PhotoURL      string `json:"photoUrl"`
		PreferredName string `json:"preferredName"`
		WorkEmail     string `json:"workEmail"`
	} `json:"employees"`
	Fields []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"fields"`
}
