package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

const BaseURL = "https://tsinglanstudent.schoolis.cn"

// Client represents an HTTP client with cookie management
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// New creates a new HTTP client with cookie jar
func New() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &Client{
		httpClient: &http.Client{Jar: jar},
		baseURL:    BaseURL,
	}, nil
}

// Get performs a GET request
func (c *Client) Get(endpoint string, queryParams map[string]string) ([]byte, error) {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if queryParams != nil {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// Post performs a POST request with JSON body
func (c *Client) Post(endpoint string, queryParams map[string]string, body interface{}) ([]byte, error) {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if queryParams != nil {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// GetJSON performs a GET request and unmarshals JSON response
func (c *Client) GetJSON(endpoint string, queryParams map[string]string, result interface{}) error {
	data, err := c.Get(endpoint, queryParams)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// PostJSON performs a POST request and unmarshals JSON response
func (c *Client) PostJSON(endpoint string, queryParams map[string]string, body interface{}, result interface{}) error {
	data, err := c.Post(endpoint, queryParams, body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}
