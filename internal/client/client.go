package client

import (
	"net/http"
	"sync"
)

// NexusClient serializes all HTTP requests to the pfSense configuration backend.
type NexusClient struct {
	mu      sync.Mutex
	baseURL string
	http    *http.Client
	token   string
}

func New(baseURL, username, password string, httpClient *http.Client) (*NexusClient, error) {
	_ = username
	_ = password
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &NexusClient{
		baseURL: baseURL,
		http:    httpClient,
	}, nil
}

func (c *NexusClient) DoRequest(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.http.Do(req)
}
