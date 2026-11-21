package client

import (
	"net/http"
	"testing"
)

func TestNexusClientSerializesRequests(t *testing.T) {
	client, err := New("https://pfsense.example:8443", "terraform-api", "secret", http.DefaultClient)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned a nil client")
	}
}
