package resource_wireguard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
						writeJSON(
							t,
							w,
							api.WireGuardSettings{Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{{Name: "wg0", Publickey: "public"}}}},
						)
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
					t.Fatalf(
						"preshared key was not retained: diagnostics=%v submitted=%#v",
						resp.Diagnostics,
						submitted.Presharedkey,
					)
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
		{
			name:        "400 still present",
			deleteCode:  http.StatusBadRequest,
			readCode:    http.StatusOK,
			stillExists: true,
			wantError:   true,
		},
		{
			name:       "400 verification failed",
			deleteCode: http.StatusBadRequest,
			readCode:   http.StatusServiceUnavailable,
			wantError:  true,
		},
		{name: "404 confirmed absent", deleteCode: http.StatusNotFound, readCode: http.StatusNotFound},
		{name: "success confirmed absent", deleteCode: http.StatusOK, readCode: http.StatusNotFound},
		{
			name:        "success still present",
			deleteCode:  http.StatusOK,
			readCode:    http.StatusOK,
			stillExists: true,
			wantError:   true,
		},
		{
			name:       "success verification failed",
			deleteCode: http.StatusOK,
			readCode:   http.StatusServiceUnavailable,
			wantError:  true,
		},
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
