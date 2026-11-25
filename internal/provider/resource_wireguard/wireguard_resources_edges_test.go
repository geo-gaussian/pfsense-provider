package resource_wireguard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
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
		{name: "update verification failed", update: true, postStatus: http.StatusOK, getStatus: http.StatusServiceUnavailable},
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
		{name: "missing", status: http.StatusOK, settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{}}}},
		{name: "invalid", status: http.StatusOK, settings: api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0"}}}}},
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
		{name: "invalid address list", status: http.StatusOK, settings: api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{{
			Publickey: "pub", Allowedips: &[]api.WGIPAddress{{}},
		}}}}},
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

func TestWireGuardWritesReportAPIAndVerificationFailures(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		resource    string
		update      bool
		writeStatus int
		readStatus  int
	}{
		{name: "tunnel create rejected", resource: "tunnel", writeStatus: http.StatusBadRequest},
		{name: "tunnel create not verified", resource: "tunnel", readStatus: http.StatusNotFound},
		{name: "tunnel create read failed", resource: "tunnel", readStatus: http.StatusServiceUnavailable},
		{name: "tunnel update rejected", resource: "tunnel", update: true, writeStatus: http.StatusBadRequest},
		{name: "tunnel update not verified", resource: "tunnel", update: true, readStatus: http.StatusNotFound},
		{name: "tunnel update read failed", resource: "tunnel", update: true, readStatus: http.StatusServiceUnavailable},
		{name: "peer create rejected", resource: "peer", writeStatus: http.StatusBadRequest},
		{name: "peer create not verified", resource: "peer", readStatus: http.StatusNotFound},
		{name: "peer create read failed", resource: "peer", readStatus: http.StatusServiceUnavailable},
		{name: "peer update rejected", resource: "peer", update: true, writeStatus: http.StatusBadRequest},
		{name: "peer update not verified", resource: "peer", update: true, readStatus: http.StatusNotFound},
		{name: "peer update read failed", resource: "peer", update: true, readStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestMethod, writePath string
			var settings api.WireGuardSettings
			var target resource.Resource
			var model any
			switch tt.resource {
			case "tunnel":
				target = NewTunnel()
				requestMethod, writePath = http.MethodPut, "/vpn/wg/tunnel"
				tunnel := &tunnelModel{
					Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
					PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
				}
				model = tunnel
				settings.Tunnels = &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0", Publickey: "public"}}}
				if tt.update {
					requestMethod, writePath = http.MethodPost, "/vpn/wg/tunnel/wg0"
				}
			case "peer":
				target = NewPeer()
				requestMethod, writePath = http.MethodPut, "/vpn/wg/peer"
				model = &peerModel{PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType())}
				settings.Peers = &api.WGPeers{Item: &[]api.WGPeer{{Publickey: "pub"}}}
				if tt.update {
					requestMethod, writePath = http.MethodPost, "/vpn/wg/peer/pub"
				}
			}
			client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method == requestMethod && req.URL.Path == writePath {
					if tt.writeStatus != 0 {
						w.WriteHeader(tt.writeStatus)
						return
					}
					writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
					return
				}
				if req.Method == http.MethodGet && req.URL.Path == "/vpn/wg" {
					if tt.readStatus != 0 {
						w.WriteHeader(tt.readStatus)
						return
					}
					writeJSON(t, w, settings)
					return
				}
				t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			})
			configurable := target.(resource.ResourceWithConfigure)
			var configureResp resource.ConfigureResponse
			configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configureResp)
			if configureResp.Diagnostics.HasError() {
				t.Fatal(configureResp.Diagnostics)
			}
			s := testResourceSchema(t, target)
			plan := testPlan(t, s, model)
			if tt.update {
				state := tfsdk.State{Schema: s}
				if diags := state.Set(ctx, model); diags.HasError() {
					t.Fatal(diags)
				}
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
				callUpdate(target, ctx, plan, state, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected update failure")
				}
				return
			}
			resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			callCreate(target, ctx, plan, &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected create failure")
			}
		})
	}
}

func TestTunnelAndPeerUpdatesRetainMaskedSecrets(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "tunnel private key",
			run: func(t *testing.T) {
				s := testResourceSchema(t, NewTunnel())
				prior := &tunnelModel{
					Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
					PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
				}
				planData := *prior
				planData.PrivateKey = types.StringNull()
				var submitted api.WGTunnel
				client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/tunnel/wg0" {
						if err := json.NewDecoder(req.Body).Decode(&submitted); err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
						return
					}
					if req.Method == http.MethodGet && req.URL.Path == "/vpn/wg" {
						writeJSON(t, w, api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0", Publickey: "public"}}}})
						return
					}
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				})
				r := &tunnelResource{resourceBase{client: client}}
				state := tfsdk.State{Schema: s}
				if diags := state.Set(ctx, prior); diags.HasError() {
					t.Fatal(diags)
				}
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
				r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, &planData), State: state}, &resp)
				if resp.Diagnostics.HasError() || submitted.Privatekey != "private" {
					t.Fatalf("private key was not retained: diagnostics=%v submitted=%q", resp.Diagnostics, submitted.Privatekey)
				}
			},
		},
		{
			name: "peer preshared key",
			run: func(t *testing.T) {
				s := testResourceSchema(t, NewPeer())
				prior := &peerModel{
					PublicKey: types.StringValue("pub"), PresharedKey: types.StringValue("secret"),
					AllowedIPs: types.ListNull(addressListType()),
				}
				planData := *prior
				planData.PresharedKey = types.StringNull()
				var submitted api.WGPeer
				client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/peer/pub" {
						if err := json.NewDecoder(req.Body).Decode(&submitted); err != nil {
							t.Error(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
						return
					}
					if req.Method == http.MethodGet && req.URL.Path == "/vpn/wg" {
						writeJSON(t, w, api.WireGuardSettings{Peers: &api.WGPeers{Item: &[]api.WGPeer{{Publickey: "pub"}}}})
						return
					}
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				})
				r := &peerResource{resourceBase{client: client}}
				state := tfsdk.State{Schema: s}
				if diags := state.Set(ctx, prior); diags.HasError() {
					t.Fatal(diags)
				}
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
				r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, &planData), State: state}, &resp)
				if resp.Diagnostics.HasError() || submitted.Presharedkey == nil || *submitted.Presharedkey != "secret" {
					t.Fatalf("preshared key was not retained: diagnostics=%v submitted=%#v", resp.Diagnostics, submitted.Presharedkey)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

func TestWireGuardDeleteVerificationBranches(t *testing.T) {
	ctx := context.Background()
	scenarios := []struct {
		name        string
		deleteCode  int
		readCode    int
		stillExists bool
		wantError   bool
	}{
		{name: "400 confirmed absent", deleteCode: http.StatusBadRequest, readCode: http.StatusNotFound},
		{name: "400 still present", deleteCode: http.StatusBadRequest, readCode: http.StatusOK, stillExists: true, wantError: true},
		{name: "400 verification failed", deleteCode: http.StatusBadRequest, readCode: http.StatusServiceUnavailable, wantError: true},
		{name: "404 confirmed absent", deleteCode: http.StatusNotFound, readCode: http.StatusNotFound},
		{name: "success confirmed absent", deleteCode: http.StatusOK, readCode: http.StatusNotFound},
		{name: "success still present", deleteCode: http.StatusOK, readCode: http.StatusOK, stillExists: true, wantError: true},
		{name: "success verification failed", deleteCode: http.StatusOK, readCode: http.StatusServiceUnavailable, wantError: true},
		{name: "server rejected delete", deleteCode: http.StatusInternalServerError, wantError: true},
	}
	for _, resourceName := range []string{"tunnel", "peer"} {
		for _, scenario := range scenarios {
			t.Run(resourceName+" "+scenario.name, func(t *testing.T) {
				var target resource.Resource
				var model any
				var deletePath string
				var settings api.WireGuardSettings
				if resourceName == "tunnel" {
					target = NewTunnel()
					deletePath = "/vpn/wg/tunnel/wg0"
					model = &tunnelModel{
						Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
						PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
					}
					if scenario.stillExists {
						settings.Tunnels = &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0", Publickey: "public"}}}
					} else {
						settings.Tunnels = &api.WGTunnels{Item: &[]api.WGTunnel{}}
					}
				} else {
					target = NewPeer()
					deletePath = "/vpn/wg/peer/pub"
					model = &peerModel{PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType())}
					if scenario.stillExists {
						settings.Peers = &api.WGPeers{Item: &[]api.WGPeer{{Publickey: "pub"}}}
					} else {
						settings.Peers = &api.WGPeers{Item: &[]api.WGPeer{}}
					}
				}
				client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
					if req.Method == http.MethodDelete && req.URL.Path == deletePath {
						if scenario.deleteCode == http.StatusOK {
							writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
						} else {
							w.WriteHeader(scenario.deleteCode)
						}
						return
					}
					if req.Method == http.MethodGet && req.URL.Path == "/vpn/wg" {
						if scenario.readCode != http.StatusOK {
							w.WriteHeader(scenario.readCode)
							return
						}
						writeJSON(t, w, settings)
						return
					}
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				})
				configurable := target.(resource.ResourceWithConfigure)
				var configureResp resource.ConfigureResponse
				configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configureResp)
				if configureResp.Diagnostics.HasError() {
					t.Fatal(configureResp.Diagnostics)
				}
				s := testResourceSchema(t, target)
				state := tfsdk.State{Schema: s}
				if diags := state.Set(ctx, model); diags.HasError() {
					t.Fatal(diags)
				}
				resp := resource.DeleteResponse{State: state}
				callDelete(target, ctx, state, &resp)
				if resp.Diagnostics.HasError() != scenario.wantError {
					t.Fatalf("unexpected delete diagnostics: %v", resp.Diagnostics)
				}
			})
		}
	}
}

func TestWireGuardWriteTransportFailures(t *testing.T) {
	ctx := context.Background()
	for _, resourceName := range []string{"settings", "tunnel", "peer"} {
		for _, operation := range []string{"create", "update", "delete"} {
			if resourceName == "settings" && operation != "update" {
				continue
			}
			t.Run(resourceName+" "+operation, func(t *testing.T) {
				var target resource.Resource
				var model any
				switch resourceName {
				case "settings":
					target = NewSettings()
					model = &settingsModel{Enable: types.BoolValue(true)}
				case "tunnel":
					target = NewTunnel()
					model = &tunnelModel{
						Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
						PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
					}
				case "peer":
					target = NewPeer()
					model = &peerModel{PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType())}
				}
				server := httptest.NewServer(http.NotFoundHandler())
				client, err := api.NewClientWithResponses(server.URL)
				if err != nil {
					t.Fatal(err)
				}
				server.Close()
				configurable := target.(resource.ResourceWithConfigure)
				var configureResp resource.ConfigureResponse
				configurable.Configure(ctx, resource.ConfigureRequest{ProviderData: client}, &configureResp)
				if configureResp.Diagnostics.HasError() {
					t.Fatal(configureResp.Diagnostics)
				}
				s := testResourceSchema(t, target)
				plan := testPlan(t, s, model)
				state := tfsdk.State{Schema: s}
				if diags := state.Set(ctx, model); diags.HasError() {
					t.Fatal(diags)
				}
				switch operation {
				case "create":
					resp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
					callCreate(target, ctx, plan, &resp)
					if !resp.Diagnostics.HasError() {
						t.Fatal("expected create transport failure")
					}
				case "update":
					resp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
					callUpdate(target, ctx, plan, state, &resp)
					if !resp.Diagnostics.HasError() {
						t.Fatal("expected update transport failure")
					}
				case "delete":
					resp := resource.DeleteResponse{State: state}
					callDelete(target, ctx, state, &resp)
					if !resp.Diagnostics.HasError() {
						t.Fatal("expected delete transport failure")
					}
				}
			})
		}
	}
}

func TestTunnelCreateAndUpdateValidatePrivateKey(t *testing.T) {
	ctx := context.Background()
	r := &tunnelResource{}
	s := testResourceSchema(t, r)
	client := testAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("API must not be called for a missing private key")
		w.WriteHeader(http.StatusInternalServerError)
	})
	r.client = client
	data := &tunnelModel{
		Name: types.StringValue("wg0"), PrivateKey: types.StringNull(),
		PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: testPlan(t, s, data)}, &createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create without private key to fail")
	}
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, data); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, data), State: state}, &updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update without private key to fail")
	}
}

func TestTunnelAndPeerCreateReportStateSchemaErrors(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name  string
		model any
		key   string
	}{
		{
			name: "tunnel",
			model: &tunnelModel{
				Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
				PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
			},
			key: "privatekey",
		},
		{
			name:  "peer",
			model: &peerModel{PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType())},
			key:   "presharedkey",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target resource.Resource
			path := "/vpn/wg/tunnel"
			switch tt.name {
			case "tunnel":
				target = NewTunnel()
			case "peer":
				target = NewPeer()
				path = "/vpn/wg/peer"
			}
			planSchema := testResourceSchema(t, target)
			stateSchema := testResourceSchema(t, target)
			stateSchema.Attributes[tt.key] = schema.BoolAttribute{Computed: true, Sensitive: true}
			client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPut || req.URL.Path != path {
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
				}
				writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
			})
			switch value := target.(type) {
			case *tunnelResource:
				value.client = client
			case *peerResource:
				value.client = client
			}
			resp := resource.CreateResponse{State: tfsdk.State{Schema: stateSchema}}
			callCreate(target, ctx, testPlan(t, planSchema, tt.model), &resp)
			if !resp.Diagnostics.HasError() {
				t.Fatal("expected incompatible state schema to fail")
			}
		})
	}
}

func TestTunnelAndPeerModelConversions(t *testing.T) {
	ctx := context.Background()
	port := 51820
	descr := "home"
	privateKey := "private"
	tunnel := tunnelModel{
		Name: types.StringValue("wg0"), Descr: types.StringValue(descr),
		Enabled: types.BoolValue(true), ListenPort: types.Int64Value(int64(port)),
		PrivateKey: types.StringValue(privateKey), PublicKey: types.StringValue("public"),
		MTU: types.StringValue("1420"),
	}
	body, diags := tunnel.toAPI(ctx)
	if diags.HasError() || body.Listenport == nil || *body.Listenport != port ||
		body.Descr == nil || *body.Descr != descr {
		t.Fatalf("unexpected tunnel API conversion: %#v, %v", body, diags)
	}
	if err := tunnel.fromAPI(ctx, &api.WGTunnel{
		Name: "wg0", Publickey: "public", Privatekey: "returned",
		Listenport: &port, Addresses: &[]api.WGIPAddress{},
	}); err != nil {
		t.Fatal(err)
	}
	if tunnel.PrivateKey.ValueString() != "returned" || tunnel.ListenPort.ValueInt64() != int64(port) {
		t.Fatalf("tunnel conversion did not retain returned values: %#v", tunnel)
	}
	for _, value := range []*api.WGTunnel{{Publickey: "public"}, {Name: "wg0"}} {
		if err := tunnel.fromAPI(ctx, value); err == nil {
			t.Fatalf("accepted incomplete tunnel: %#v", value)
		}
	}
	if err := tunnel.fromAPI(ctx, &api.WGTunnel{
		Name: "wg0", Publickey: "public", Addresses: &[]api.WGIPAddress{{}},
	}); err == nil {
		t.Fatal("accepted a tunnel address without a value")
	}

	preshared := "secret"
	peer := peerModel{
		PublicKey: types.StringValue("pub"), Enabled: types.BoolValue(true),
		Tunnel: types.StringValue("wg0"), Descr: types.StringValue(descr),
		Endpoint: types.StringValue("peer.example.test"), Port: types.StringValue("51820"),
		PersistentKeepalive: types.StringValue("25"), PresharedKey: types.StringValue(preshared),
		AllowedIPs: types.ListNull(addressListType()),
	}
	peerBody, peerDiags := peer.toAPI(ctx)
	if peerDiags.HasError() || peerBody.Presharedkey == nil || *peerBody.Presharedkey != preshared {
		t.Fatalf("unexpected peer API conversion: %#v, %v", peerBody, peerDiags)
	}
	if err := peer.fromAPI(ctx, &api.WGPeer{
		Publickey: "pub", Presharedkey: &preshared, Allowedips: &[]api.WGIPAddress{},
	}); err != nil {
		t.Fatal(err)
	}
	if peer.PresharedKey.ValueString() != preshared {
		t.Fatalf("peer conversion did not retain returned secret: %#v", peer)
	}
	if err := peer.fromAPI(ctx, &api.WGPeer{}); err == nil {
		t.Fatal("accepted a peer without a public key")
	}
	if err := peer.fromAPI(ctx, &api.WGPeer{
		Publickey: "pub", Allowedips: &[]api.WGIPAddress{{}},
	}); err == nil {
		t.Fatal("accepted a peer allowed IP without a value")
	}

	if _, err := addressListValue(ctx, nil); err != nil {
		t.Fatalf("nil address list returned an error: %v", err)
	}
	if intValue(&port).IsNull() || intValue(&port).IsUnknown() {
		t.Fatal("non-nil integer was not converted")
	}
}

func TestTunnelAndPeerToAPIRejectInvalidAddressLists(t *testing.T) {
	ctx := context.Background()
	wrongTypeList := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("not an object")})
	tunnel := tunnelModel{Name: types.StringValue("wg0"), Addresses: wrongTypeList}
	if _, diags := tunnel.toAPI(ctx); !diags.HasError() {
		t.Fatal("expected invalid tunnel addresses to be rejected")
	}
	peer := peerModel{PublicKey: types.StringValue("pub"), AllowedIPs: wrongTypeList}
	if _, diags := peer.toAPI(ctx); !diags.HasError() {
		t.Fatal("expected invalid peer allowed IPs to be rejected")
	}
}

func TestWireGuardResourcesRejectMalformedPlanAndState(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name     string
		resource resource.Resource
		model    any
	}{
		{
			name:     "settings",
			resource: NewSettings(),
			model:    &settingsModel{Enable: types.BoolValue(true)},
		},
		{
			name:     "tunnel",
			resource: NewTunnel(),
			model: &tunnelModel{
				Name: types.StringValue("wg0"), PrivateKey: types.StringValue("secret"),
				PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
			},
		},
		{
			name:     "peer",
			resource: NewPeer(),
			model: &peerModel{
				PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType()),
			},
		},
	}
	invalid := tftypes.NewValue(tftypes.String, "invalid")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testResourceSchema(t, tt.resource)
			badPlan := tfsdk.Plan{Schema: s, Raw: invalid}
			create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
			callCreate(tt.resource, ctx, badPlan, &create)
			if !create.Diagnostics.HasError() {
				t.Fatal("expected malformed plan to be rejected")
			}

			plan := testPlan(t, s, tt.model)
			state := tfsdk.State{Schema: s, Raw: invalid}
			read := resource.ReadResponse{State: tfsdk.State{Schema: s}}
			callRead(tt.resource, ctx, state, &read)
			if !read.Diagnostics.HasError() {
				t.Fatal("expected malformed state to be rejected by Read")
			}

			update := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
			updatePlan := plan
			if tt.name == "settings" {
				updatePlan = badPlan
			}
			callUpdate(tt.resource, ctx, updatePlan, state, &update)
			if !update.Diagnostics.HasError() {
				t.Fatal("expected malformed plan or state to be rejected by Update")
			}

			if tt.name != "settings" {
				deleteResponse := resource.DeleteResponse{State: tfsdk.State{Schema: s}}
				callDelete(tt.resource, ctx, state, &deleteResponse)
				if !deleteResponse.Diagnostics.HasError() {
					t.Fatal("expected malformed state to be rejected by Delete")
				}
			}
		})
	}
}

func callCreate(r resource.Resource, ctx context.Context, plan tfsdk.Plan, resp *resource.CreateResponse) {
	switch value := r.(type) {
	case *settingsResource:
		value.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	case *tunnelResource:
		value.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	case *peerResource:
		value.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	}
}

func callRead(r resource.Resource, ctx context.Context, state tfsdk.State, resp *resource.ReadResponse) {
	switch value := r.(type) {
	case *settingsResource:
		value.Read(ctx, resource.ReadRequest{State: state}, resp)
	case *tunnelResource:
		value.Read(ctx, resource.ReadRequest{State: state}, resp)
	case *peerResource:
		value.Read(ctx, resource.ReadRequest{State: state}, resp)
	}
}

func callUpdate(r resource.Resource, ctx context.Context, plan tfsdk.Plan, state tfsdk.State, resp *resource.UpdateResponse) {
	switch value := r.(type) {
	case *settingsResource:
		value.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
	case *tunnelResource:
		value.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
	case *peerResource:
		value.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
	}
}

func callDelete(r resource.Resource, ctx context.Context, state tfsdk.State, resp *resource.DeleteResponse) {
	switch value := r.(type) {
	case *settingsResource:
		value.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	case *tunnelResource:
		value.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	case *peerResource:
		value.Delete(ctx, resource.DeleteRequest{State: state}, resp)
	}
}
