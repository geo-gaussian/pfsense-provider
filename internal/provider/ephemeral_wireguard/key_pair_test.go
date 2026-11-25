package ephemeral_wireguard

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestKeyPairOpenGeneratesSensitiveEphemeralValues(t *testing.T) {
	privateKey := []byte{1, 2, 3}
	clampedKey := []byte{4, 5, 6}
	publicKey := []byte{7, 8, 9}
	wasClamped := true
	responseBody, err := json.Marshal(api.WireGuardKeys{
		Privkey:        &privateKey,
		PrivkeyClamped: &clampedKey,
		Pubkey:         &publicKey,
		WasClamped:     &wasClamped,
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg/genkeys" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := New().(*keyPairResource)
	var configureResp ephemeral.ConfigureResponse
	r.Configure(context.Background(), ephemeral.ConfigureRequest{ProviderData: client}, &configureResp)
	if configureResp.Diagnostics.HasError() {
		t.Fatal(configureResp.Diagnostics)
	}

	var schemaResp ephemeral.SchemaResponse
	r.Schema(context.Background(), ephemeral.SchemaRequest{}, &schemaResp)
	for _, name := range []string{"private_key", "private_key_clamped"} {
		attribute, ok := schemaResp.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Sensitive {
			t.Fatalf("%s must be a sensitive string", name)
		}
	}

	resp := ephemeral.OpenResponse{
		Result: tfsdk.EphemeralResultData{Schema: schemaResp.Schema},
	}
	r.Open(context.Background(), ephemeral.OpenRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var result keyPairModel
	if diags := resp.Result.Get(context.Background(), &result); diags.HasError() {
		t.Fatal(diags)
	}
	if result.PrivateKey.ValueString() != base64.StdEncoding.EncodeToString(privateKey) ||
		result.PrivateKeyClamped.ValueString() != base64.StdEncoding.EncodeToString(clampedKey) ||
		result.PublicKey.ValueString() != base64.StdEncoding.EncodeToString(publicKey) ||
		!result.WasClamped.ValueBool() {
		t.Fatalf("unexpected generated key pair: %#v", result)
	}
}

func TestKeyPairOpenRejectsMissingKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pubkey":"BwgJ"}`))
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := &keyPairResource{client: client}
	var schemaResp ephemeral.SchemaResponse
	r.Schema(context.Background(), ephemeral.SchemaRequest{}, &schemaResp)
	resp := ephemeral.OpenResponse{
		Result: tfsdk.EphemeralResultData{Schema: schemaResp.Schema},
	}
	r.Open(context.Background(), ephemeral.OpenRequest{}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an error when the API omits private key material")
	}
}
