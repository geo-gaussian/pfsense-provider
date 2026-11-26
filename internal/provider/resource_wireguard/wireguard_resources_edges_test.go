package resource_wireguard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWireGuardResourceConstructorsMetadataAndConfigure(t *testing.T) {
	ctx := context.Background()
	client := testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	tests := []struct {
		name     string
		resource resource.Resource
		typeName string
		idPath   string
		id       string
	}{
		{"settings", NewSettings(), "pfsense_wireguard_settings", "id", settingsID},
		{"tunnel", NewTunnel(), "pfsense_wireguard_tunnel", "name", "wg0"},
		{"peer", NewPeer(), "pfsense_wireguard_peer", "publickey", "pub"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var metadata resource.MetadataResponse
			tt.resource.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, &metadata)
			if metadata.TypeName != tt.typeName {
				t.Fatalf("type name = %q, want %q", metadata.TypeName, tt.typeName)
			}

			configurable := tt.resource.(resource.ResourceWithConfigure)
			for _, providerData := range []any{nil, "invalid", client} {
				var response resource.ConfigureResponse
				configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: providerData}, &response)
				if response.Diagnostics.HasError() != (providerData == "invalid") {
					t.Fatalf("unexpected configure diagnostics for %#v: %v", providerData, response.Diagnostics)
				}
			}

			importable := tt.resource.(resource.ResourceWithImportState)
			s := testResourceSchema(t, tt.resource)
			state := tfsdk.State{Schema: s}
			switch tt.name {
			case "settings":
				if diags := state.Set(ctx, &settingsModel{}); diags.HasError() {
					t.Fatal(diags)
				}
			case "tunnel":
				if diags := state.Set(ctx, &tunnelModel{Addresses: types.ListNull(addressListType())}); diags.HasError() {
					t.Fatal(diags)
				}
			case "peer":
				if diags := state.Set(ctx, &peerModel{AllowedIPs: types.ListNull(addressListType())}); diags.HasError() {
					t.Fatal(diags)
				}
			}
			response := resource.ImportStateResponse{State: state}
			importable.ImportState(ctx, resource.ImportStateRequest{ID: tt.id}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			var imported string
			if diags := response.State.GetAttribute(ctx, path.Root(tt.idPath), &imported); diags.HasError() {
				t.Fatal(diags)
			}
			if imported != tt.id {
				t.Fatalf("imported %s = %q, want %q", tt.idPath, imported, tt.id)
			}
		})
	}
}

func TestSettingsReadHandlesAPIResponses(t *testing.T) {
	ctx := context.Background()
	oneConfig := []api.WGConfig{{Enable: boolPointer(types.BoolValue(true))}}
	multipleConfigs := []api.WGConfig{{}, {}}
	tests := []struct {
		name       string
		status     int
		body       any
		wantError  bool
		wantAbsent bool
	}{
		{
			name:   "found",
			status: http.StatusOK,
			body:   api.WireGuardSettings{Config: &oneConfig},
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			wantAbsent: true,
		},
		{
			name:      "server error",
			status:    http.StatusServiceUnavailable,
			wantError: true,
		},
		{
			name:      "missing settings",
			status:    http.StatusOK,
			body:      api.WireGuardSettings{},
			wantError: true,
		},
		{
			name:      "empty settings",
			status:    http.StatusOK,
			body:      api.WireGuardSettings{Config: &[]api.WGConfig{}},
			wantError: true,
		},
		{
			name:      "multiple settings",
			status:    http.StatusOK,
			body:      api.WireGuardSettings{Config: &multipleConfigs},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg" {
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if tt.body != nil {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(tt.status)
				if tt.body != nil {
					if err := json.NewEncoder(w).Encode(tt.body); err != nil {
						t.Error(err)
					}
				}
			})
			r := &settingsResource{resourceBase{client: client}}
			s := testResourceSchema(t, r)
			state := tfsdk.State{Schema: s}
			if diags := state.Set(ctx, &settingsModel{ID: types.StringValue(settingsID)}); diags.HasError() {
				t.Fatal(diags)
			}
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if response.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected read diagnostics: %v", response.Diagnostics)
			}
			if response.State.Raw.IsNull() != tt.wantAbsent {
				t.Fatalf("state absent = %t, want %t", response.State.Raw.IsNull(), tt.wantAbsent)
			}
		})
	}
}

func TestSettingsWriteAndVerificationFailures(t *testing.T) {
	ctx := context.Background()
	s := testResourceSchema(t, NewSettings())
	data := &settingsModel{Enable: types.BoolValue(true)}
	plan := testPlan(t, s, data)
	tests := []struct {
		name       string
		getStatus  int
		postStatus int
		update     bool
	}{
		{name: "create write rejected", postStatus: http.StatusBadRequest},
		{name: "create verification missing", postStatus: http.StatusOK, getStatus: http.StatusNotFound},
		{name: "create verification failed", postStatus: http.StatusOK, getStatus: http.StatusServiceUnavailable},
		{name: "update write rejected", update: true, postStatus: http.StatusBadRequest},
		{name: "update verification missing", update: true, postStatus: http.StatusOK, getStatus: http.StatusNotFound},
		{
			name:       "update verification failed",
			update:     true,
			postStatus: http.StatusOK,
			getStatus:  http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/settings" {
					if tt.postStatus != http.StatusOK {
						w.WriteHeader(tt.postStatus)
						return
					}
					writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
					return
				}
				if req.Method == http.MethodGet && req.URL.Path == "/vpn/wg" {
					if tt.getStatus != http.StatusOK {
						w.WriteHeader(tt.getStatus)
						return
					}
					writeJSON(t, w, api.WireGuardSettings{Config: &[]api.WGConfig{{Enable: boolPointer(types.BoolValue(true))}}})
					return
				}
				t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			})
			r := &settingsResource{resourceBase{client: client}}
			state := tfsdk.State{Schema: s}
			if diags := state.Set(ctx, data); diags.HasError() {
				t.Fatal(diags)
			}
			if tt.update {
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update failure")
				}
				return
			}
			resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected create failure")
			}
		})
	}

	t.Run("create state write failure", func(t *testing.T) {
		client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
			if req.Method != http.MethodPost || req.URL.Path != "/vpn/wg/settings" {
				t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		})
		r := &settingsResource{resourceBase{client: client}}
		resp := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema{
			Attributes: map[string]schema.Attribute{
				"id":     schema.StringAttribute{Computed: true},
				"enable": schema.StringAttribute{Computed: true},
			},
		}}}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected incompatible state schema to fail")
		}
	})

	t.Run("write transport failure", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		client, err := api.NewClientWithResponses(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		server.Close()
		r := &settingsResource{resourceBase{client: client}}
		resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected a transport failure")
		}
	})

	for _, status := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status)+" settings verification", func(t *testing.T) {
			r := &settingsResource{resourceBase{client: testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})}}
			var diags diag.Diagnostics
			r.readSettingsIntoState(ctx, &settingsModel{}, &diags)
			if !diags.HasError() {
				t.Fatal("expected settings verification failure")
			}
		})
	}
}

func TestTunnelAndPeerReadResponses(t *testing.T) {
	ctx := context.Background()
	tunnelSchema := testResourceSchema(t, NewTunnel())
	peerSchema := testResourceSchema(t, NewPeer())
	tunnelCases := []struct {
		name        string
		status      int
		settings    api.WireGuardSettings
		wantError   bool
		wantAbsent  bool
		wantPrivate string
	}{
		{
			name: "found with masked key", status: http.StatusOK,
			settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{
				Name: "wg0", Publickey: "public",
			}}}},
			wantPrivate: "private",
		},
		{name: "not found", status: http.StatusNotFound, wantAbsent: true},
		{name: "server error", status: http.StatusServiceUnavailable, wantError: true},
		{name: "omitted tunnel list", status: http.StatusOK, settings: api.WireGuardSettings{}, wantError: true},
		{
			name: "invalid tunnel", status: http.StatusOK, wantError: true,
			settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{
				Name: "wg0",
			}}}},
		},
	}
	for _, tt := range tunnelCases {
		t.Run("tunnel "+tt.name, func(t *testing.T) {
			client := testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, tt.settings)
			})
			r := &tunnelResource{resourceBase{client: client}}
			state := tfsdk.State{Schema: tunnelSchema}
			initial := &tunnelModel{
				Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
				PublicKey: types.StringValue("old-public"), Addresses: types.ListNull(addressListType()),
			}
			if diags := state.Set(ctx, initial); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected tunnel read diagnostics: %v", resp.Diagnostics)
			}
			if resp.State.Raw.IsNull() != tt.wantAbsent {
				t.Fatalf("tunnel state absent = %t, want %t", resp.State.Raw.IsNull(), tt.wantAbsent)
			}
			if tt.wantPrivate != "" {
				var actual tunnelModel
				if diags := resp.State.Get(ctx, &actual); diags.HasError() {
					t.Fatal(diags)
				}
				if actual.PrivateKey.ValueString() != tt.wantPrivate {
					t.Fatalf("masked private key = %q, want retained key", actual.PrivateKey.ValueString())
				}
			}
		})
	}

	peerCases := []struct {
		name       string
		status     int
		settings   api.WireGuardSettings
		wantError  bool
		wantAbsent bool
	}{
		{
			name: "found with masked key", status: http.StatusOK,
			settings: api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{{
				Publickey: "pub",
			}}}},
		},
		{
			name: "invalid allowed IP", status: http.StatusOK, wantError: true,
			settings: api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{{
				Publickey: "pub", Allowedips: &[]api.WGIPAddress{{}},
			}}}},
		},
		{name: "not found", status: http.StatusNotFound, wantAbsent: true},
		{name: "server error", status: http.StatusServiceUnavailable, wantError: true},
		{name: "omitted peer list", status: http.StatusOK, settings: api.WireGuardSettings{}, wantError: true},
	}
	for _, tt := range peerCases {
		t.Run("peer "+tt.name, func(t *testing.T) {
			client := testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, tt.settings)
			})
			r := &peerResource{resourceBase{client: client}}
			state := tfsdk.State{Schema: peerSchema}
			initial := &peerModel{
				PublicKey: types.StringValue("pub"), PresharedKey: types.StringValue("secret"),
				AllowedIPs: types.ListNull(addressListType()),
			}
			if diags := state.Set(ctx, initial); diags.HasError() {
				t.Fatal(diags)
			}
			resp := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Fatalf("unexpected peer read diagnostics: %v", resp.Diagnostics)
			}
			if resp.State.Raw.IsNull() != tt.wantAbsent {
				t.Fatalf("peer state absent = %t, want %t", resp.State.Raw.IsNull(), tt.wantAbsent)
			}
			if tt.name == "found with masked key" {
				var actual peerModel
				if diags := resp.State.Get(ctx, &actual); diags.HasError() {
					t.Fatal(diags)
				}
				if actual.PresharedKey.ValueString() != "secret" {
					t.Fatalf("masked preshared key = %q, want retained key", actual.PresharedKey.ValueString())
				}
			}
		})
	}
}

func TestWireGuardReadHelpersAndConversionEdges(t *testing.T) {
	ctx := context.Background()

	server := httptest.NewServer(http.NotFoundHandler())
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	settings := &settingsResource{resourceBase{client: client}}
	if _, _, err := settings.getSettings(ctx); err == nil {
		t.Fatal("expected settings transport failure")
	}
	if _, _, err := settings.resourceBase.getAll(ctx); err == nil {
		t.Fatal("expected configuration transport failure")
	}

	description := "address description"
	addresses, err := addressListValue(ctx, &[]api.WGIPAddress{{Address: "192.0.2.1/32", Descr: &description}})
	if err != nil || addresses.IsNull() {
		t.Fatalf("valid API address list conversion failed: %v", err)
	}
	if _, err := addressListValue(ctx, &[]api.WGIPAddress{{}}); err == nil {
		t.Fatal("accepted an API address without a value")
	}

	peer := peerModel{PresharedKey: types.StringValue("retained")}
	emptySecret := ""
	if err := peer.fromAPI(ctx, &api.WGPeer{Publickey: "pub", Presharedkey: &emptySecret}); err != nil {
		t.Fatal(err)
	}
	if peer.PresharedKey.ValueString() != "retained" {
		t.Fatalf("empty API secret replaced prior state: %q", peer.PresharedKey.ValueString())
	}

	for _, tt := range []struct {
		name     string
		settings api.WireGuardSettings
		status   int
	}{
		{
			name: "missing", status: http.StatusOK,
			settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{}}},
		},
		{
			name: "invalid", status: http.StatusOK,
			settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0"}}}},
		},
		{name: "api error", status: http.StatusServiceUnavailable},
	} {
		t.Run("tunnel verification "+tt.name, func(t *testing.T) {
			r := &tunnelResource{resourceBase{client: testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, tt.settings)
			})}}
			var diags diag.Diagnostics
			data := &tunnelModel{PrivateKey: types.StringValue("private")}
			r.readTunnelIntoState(ctx, "wg0", data, &diags)
			if !diags.HasError() {
				t.Fatal("expected tunnel verification error")
			}
		})
	}

	for _, tt := range []struct {
		name     string
		settings api.WireGuardSettings
		status   int
	}{
		{name: "missing", status: http.StatusOK, settings: api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{}}}},
		{
			name: "invalid address list", status: http.StatusOK,
			settings: api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{{
				Publickey: "pub", Allowedips: &[]api.WGIPAddress{{}},
			}}}},
		},
		{name: "api error", status: http.StatusServiceUnavailable},
	} {
		t.Run("peer verification "+tt.name, func(t *testing.T) {
			r := &peerResource{resourceBase{client: testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if tt.status != http.StatusOK {
					w.WriteHeader(tt.status)
					return
				}
				writeJSON(t, w, tt.settings)
			})}}
			var diags diag.Diagnostics
			data := &peerModel{}
			r.readPeerIntoState(ctx, "pub", data, &diags)
			if !diags.HasError() {
				t.Fatal("expected peer verification error")
			}
		})
	}
}
