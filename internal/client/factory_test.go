package client

import (
	"errors"
	"net/http/cookiejar"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
)

func TestNewReportsFactoryFailures(t *testing.T) {
	want := errors.New("factory failure")
	tests := []struct {
		name   string
		jar    func(*cookiejar.Options) (*cookiejar.Jar, error)
		client func(string, ...api.ClientOption) (*api.ClientWithResponses, error)
	}{
		{
			name: "cookie jar",
			jar: func(*cookiejar.Options) (*cookiejar.Jar, error) {
				return nil, want
			},
			client: api.NewClientWithResponses,
		},
		{
			name: "API client",
			jar:  cookiejar.New,
			client: func(string, ...api.ClientOption) (*api.ClientWithResponses, error) {
				return nil, want
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newWithFactories(
				"https://example.test",
				"test-user",
				strings.Repeat("x", 12),
				false,
				tt.jar,
				tt.client,
			)
			if client != nil || !errors.Is(err, want) {
				t.Fatalf("client = %v, error = %v", client, err)
			}
		})
	}
}
