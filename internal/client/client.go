package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
)

// NexusClient serializes requests and maintains the controller session.
type NexusClient struct {
	mu        sync.Mutex
	requestMu sync.Mutex
	http      *http.Client
	auth      *api.ClientWithResponses
	username  string
	password  string
	token     string
	renewAt   time.Time
}

func New(baseURL, username, password string, insecure bool) (*NexusClient, error) {
	return newWithFactories(baseURL, username, password, insecure, cookiejar.New, api.NewClientWithResponses)
}

func newWithFactories(
	baseURL, username, password string,
	insecure bool,
	newJar func(*cookiejar.Options) (*cookiejar.Jar, error),
	newAPI func(string, ...api.ClientOption) (*api.ClientWithResponses, error),
) (*NexusClient, error) {
	if err := validateConfiguration(baseURL, username, password); err != nil {
		return nil, err
	}
	jar, err := newJar(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: insecure,
	} // Explicit opt-out for test environments.
	c := &NexusClient{
		http: &http.Client{
			Transport: transport,
			Jar:       jar,
			Timeout:   40 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		username: base64.StdEncoding.EncodeToString([]byte(username)),
		password: base64.StdEncoding.EncodeToString([]byte(password)),
	}
	c.auth, err = newAPI(strings.TrimRight(baseURL, "/")+"/api", api.WithHTTPClient(rawClient{c}))
	if err != nil {
		return nil, fmt.Errorf("configure API client: %w", err)
	}
	return c, nil
}

func validateConfiguration(baseURL, username, password string) error {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		(u.Path != "" && u.Path != "/") {
		return fmt.Errorf("controller URL must be an HTTPS origin without a path, credentials, query, or fragment")
	}
	if username == "" || password == "" {
		return fmt.Errorf("username and password must not be empty")
	}
	return nil
}

// DoRequest is the single transport entry point for generated API clients.
func (c *NexusClient) DoRequest(req *http.Request) (*http.Response, error) {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	return c.http.Do(req)
}

type rawClient struct{ client *NexusClient }

func (r rawClient) Do(req *http.Request) (*http.Response, error) {
	return r.client.DoRequest(req)
}

// Do authenticates and serializes each API request.
func (c *NexusClient) Do(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureSession(req.Context()); err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.DoRequest(req)
}

func (c *NexusClient) ensureSession(ctx context.Context) error {
	if c.token == "" {
		resp, err := c.auth.LoginWithResponse(ctx, api.LoginCredentials{
			Username: c.username,
			Password: c.password,
		})
		if err != nil {
			return fmt.Errorf("authenticate with Nexus: %w", err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil || resp.JSON200.Token == nil ||
			*resp.JSON200.Token == "" {
			return fmt.Errorf("Nexus authentication failed (HTTP %d or missing access token)", resp.StatusCode())
		}
		c.token = *resp.JSON200.Token
		c.renewAt = time.Now().Add(4 * time.Minute)
		return nil
	}
	if time.Now().Before(c.renewAt) {
		return nil
	}
	resp, err := c.auth.RefreshAccessTokenWithResponse(
		ctx,
		api.RefreshTokenParam{Username: &c.username},
		func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+c.token)
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("refresh Nexus session: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil || resp.JSON200.Token == nil ||
		*resp.JSON200.Token == "" {
		return fmt.Errorf("Nexus session refresh failed (HTTP %d or missing access token)", resp.StatusCode())
	}
	c.token = *resp.JSON200.Token
	c.renewAt = time.Now().Add(4 * time.Minute)
	return nil
}
