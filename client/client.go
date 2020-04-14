package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Unbabel/replicant/api"
	"github.com/Unbabel/replicant/transaction"
)

// Config for replicant client
type Config struct {
	URL                string
	Username           string
	Password           string
	Timeout            time.Duration
	InsecureSkipVerify bool
}

// Client for replicant api
type Client struct {
	http   *http.Client
	config Config
}

// New creates a new client
func New(c Config) (client *Client, err error) {
	client = &Client{}
	client.config = c
	transport := &http.Transport{}
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify}
	client.http = &http.Client{Transport: transport}
	client.http.Timeout = c.Timeout

	return client, nil
}

// Run the given transaction definition on the server
func (c *Client) Run(t transaction.Config) (r transaction.Result, err error) {

	buf, err := json.Marshal(&t)
	if err != nil {
		return r, fmt.Errorf("client: error marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.config.URL+api.EndpointRun, bytes.NewReader(buf))
	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return r, fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return r, fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return r, fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return r, fmt.Errorf("client: server error: %s", ar.Error)
	}

	if len(ar.Results) == 0 {
		return r, fmt.Errorf("client: server error: %#v", ar)
	}

	return ar.Results[0], nil
}

// RunByName a managed transaction on the server
func (c *Client) RunByName(name string) (r transaction.Result, err error) {
	req, err := http.NewRequest(http.MethodPost, c.config.URL+api.EndpointRun+"/"+name, nil)
	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return r, fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return r, fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return r, fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return r, fmt.Errorf("client: server error: %s", ar.Error)
	}

	if len(ar.Results) == 0 {
		return r, fmt.Errorf("client: server error: %#v", ar)
	}

	return ar.Results[0], nil
}

// GetTransaction fetches the given transaction definition from server
func (c *Client) GetTransaction(name string) (t transaction.Config, err error) {
	ts, err := c.getTransactions(name)
	if err != nil {
		return t, err
	}

	if len(ts) == 0 {
		return t, fmt.Errorf("client: no transaction with name %s found", name)
	}

	return ts[0], nil
}

// GetTransactions fetches all transactions from server
func (c *Client) GetTransactions() (t []transaction.Config, err error) {
	return c.getTransactions("")
}

func (c *Client) getTransactions(name string) (t []transaction.Config, err error) {

	var req *http.Request
	switch name != "" {
	case true:
		req, err = http.NewRequest(http.MethodGet, c.config.URL+api.EndpointTransaction+"/"+name, nil)
	case false:
		req, err = http.NewRequest(http.MethodGet, c.config.URL+api.EndpointTransaction, nil)
	}

	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return nil, fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return nil, fmt.Errorf("client: server error: %s", ar.Error)
	}

	return ar.Transactions, nil
}

// GetResult fetches the latest result for the given transaction from the server
func (c *Client) GetResult(name string) (r transaction.Result, err error) {
	rs, err := c.getResults(name)
	if err != nil {
		return r, err
	}

	if len(rs) == 0 {
		return r, fmt.Errorf("client: no transaction with name %s found", name)
	}

	return rs[0], nil
}

// GetResults fetches the latest result for all transactions from the server
func (c *Client) GetResults() (t []transaction.Result, err error) {
	return c.getResults("")
}

func (c *Client) getResults(name string) (t []transaction.Result, err error) {

	var req *http.Request
	switch name != "" {
	case true:
		req, err = http.NewRequest(http.MethodGet, c.config.URL+api.EndpointResult+"/"+name, nil)
	case false:
		req, err = http.NewRequest(http.MethodGet, c.config.URL+api.EndpointResult, nil)
	}

	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return nil, fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return nil, fmt.Errorf("client: server error: %s", ar.Error)
	}

	return ar.Results, nil
}

// Add the given transaction definition
func (c *Client) Add(t transaction.Config) (err error) {

	buf, err := json.Marshal(&t)
	if err != nil {
		return fmt.Errorf("client: error marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.config.URL+api.EndpointTransaction, bytes.NewReader(buf))
	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return fmt.Errorf("client: server error: %s", ar.Error)
	}

	return nil
}

// Delete a managed transaction
func (c *Client) Delete(name string) (err error) {
	if name == "" {
		return fmt.Errorf("client: must specify transaction name")
	}

	req, err := http.NewRequest(http.MethodDelete, c.config.URL+api.EndpointTransaction+"/"+name, nil)
	req.Header.Add("Content-Type", "application/json")
	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("client: error sending request: %w", err)
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("client: error reading response: %w", err)
	}

	var ar api.Result
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return fmt.Errorf("client: error unmarshaling response: %w", err)
	}

	if ar.Error != "" {
		return fmt.Errorf("client: server error: %s", ar.Error)
	}

	return nil
}
