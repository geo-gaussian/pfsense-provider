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

func TestKeyPairMetadataConfigureAndOpenErrors(t *testing.T) {
	ctx := context.Background()
	r := New().(*keyPairResource)

	var metadata ephemeral.MetadataResponse
	r.Metadata(ctx, ephemeral.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
	if metadata.TypeName != "pfsense_wireguard_key_pair" {
		t.Fatalf("unexpected ephemeral resource type name %q", metadata.TypeName)
	}

	for _, providerData := range []any{nil, "invalid"} {
		var response ephemeral.ConfigureResponse
		r.Configure(ctx, ephemeral.ConfigureRequest{ProviderData: providerData}, &response)
		if response.Diagnostics.HasError() != (providerData != nil) {
			t.Fatalf("unexpected Configure diagnostics for %#v: %v", providerData, response.Diagnostics)
		}
	}

	var schemaResp ephemeral.SchemaResponse
	r.Schema(ctx, ephemeral.SchemaRequest{}, &schemaResp)
	t.Run("client missing", func(t *testing.T) {
		response := openResponse(schemaResp.Schema)
		r.Open(ctx, ephemeral.OpenRequest{}, &response)
		if !response.Diagnostics.HasError() {
			t.Fatal("expected an error without a configured client")
		}
	})

	t.Run("transport error", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		client, err := api.NewClientWithResponses(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		server.Close()
		configured := &keyPairResource{client: client}
		response := openResponse(schemaResp.Schema)
		configured.Open(ctx, ephemeral.OpenRequest{}, &response)
		if !response.Diagnostics.HasError() {
			t.Fatal("expected an API transport error")
		}
	})

	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "rejected status", status: http.StatusServiceUnavailable},
		{name: "null response", status: http.StatusOK, body: "null"},
		{name: "empty private key", status: http.StatusOK, body: `{"privkey":[],"pubkey":[1]}`},
		{name: "empty public key", status: http.StatusOK, body: `{"privkey":[1],"pubkey":[]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := keyPairClient(t, tt.status, tt.body)
			configured := &keyPairResource{client: client}
			response := openResponse(schemaResp.Schema)
			configured.Open(ctx, ephemeral.OpenRequest{}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("expected invalid API response to fail")
			}
		})
	}

	t.Run("optional fields omitted", func(t *testing.T) {
		client := keyPairClient(t, http.StatusOK, `{"privkey":[1,2],"pubkey":[3,4]}`)
		configured := &keyPairResource{client: client}
		response := openResponse(schemaResp.Schema)
		configured.Open(ctx, ephemeral.OpenRequest{}, &response)
		if response.Diagnostics.HasError() {
			t.Fatal(response.Diagnostics)
		}
		var result keyPairModel
		if diags := response.Result.Get(ctx, &result); diags.HasError() {
			t.Fatal(diags)
		}
		if !result.PrivateKeyClamped.IsNull() || !result.WasClamped.IsNull() {
			t.Fatalf("omitted optional values should remain null: %#v", result)
		}
	})

	t.Run("result schema mismatch", func(t *testing.T) {
		client := keyPairClient(t, http.StatusOK, `{"privkey":[1],"pubkey":[2]}`)
		configured := &keyPairResource{client: client}
		response := openResponse(schemaResp.Schema)
		response.Result.Schema = schema.Schema{Attributes: map[string]schema.Attribute{
			"private_key":         schema.BoolAttribute{Computed: true, Sensitive: true},
			"private_key_clamped": schema.StringAttribute{Computed: true, Sensitive: true},
			"public_key":          schema.StringAttribute{Computed: true},
			"was_clamped":         schema.BoolAttribute{Computed: true},
		}}
		configured.Open(ctx, ephemeral.OpenRequest{}, &response)
		if !response.Diagnostics.HasError() {
			t.Fatal("expected incompatible result schema to fail")
		}
	})
}

func openResponse(s schema.Schema) ephemeral.OpenResponse {
	return ephemeral.OpenResponse{Result: tfsdk.EphemeralResultData{Schema: s}}
}

func keyPairClient(t *testing.T, status int, body string) *api.ClientWithResponses {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(server.Close)
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client
}
