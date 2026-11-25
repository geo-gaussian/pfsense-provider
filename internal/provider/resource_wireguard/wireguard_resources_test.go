package resource_wireguard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func testAPIClient(t *testing.T, handler http.HandlerFunc) *api.ClientWithResponses {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func testResourceSchema(t *testing.T, value resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	value.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}

func testPlan(t *testing.T, schema schema.Schema, value any) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), value); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}

func TestSettingsLifecycle(t *testing.T) {
	ctx := context.Background()
	stored := api.WGConfig{Enable: boolPointer(types.BoolValue(true))}
	client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/settings":
			if err := json.NewDecoder(req.Body).Decode(&stored); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodGet && req.URL.Path == "/vpn/wg":
			configs := []api.WGConfig{stored}
			tunnels := api.WGTunnels{Item: &[]api.WGTunnel{}}
			peers := api.WGPeers{Item: &[]api.WGPeer{}}
			writeJSON(t, w, api.WireGuardSettings{Config: &configs, Tunnels: &tunnels, Peers: &peers})
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := &settingsResource{}
	r.client = client
	s := testResourceSchema(t, r)
	data := settingsModel{Enable: types.BoolValue(true)}
	plan := testPlan(t, s, &data)
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	var created settingsModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatal(diags)
	}
	if created.ID.ValueString() != settingsID || !created.Enable.ValueBool() {
		t.Fatalf("unexpected created settings state: %#v", created)
	}

	data.Enable = types.BoolValue(false)
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, &data), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if stored.Enable == nil || *stored.Enable {
		t.Fatalf("settings were not updated: %#v", stored)
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
}

func TestTunnelLifecycleRetainsMaskedPrivateKey(t *testing.T) {
	ctx := context.Background()
	var stored *api.WGTunnel
	client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/vpn/wg/tunnel":
			var body api.WGTunnel
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/tunnel/wg0":
			var body api.WGTunnel
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodDelete && req.URL.Path == "/vpn/wg/tunnel/wg0":
			stored = nil
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodGet && req.URL.Path == "/vpn/wg":
			tunnels := make([]api.WGTunnel, 0)
			if stored != nil {
				masked := *stored
				masked.Privatekey = ""
				tunnels = append(tunnels, masked)
			}
			wgTunnels := api.WGTunnels{Item: &tunnels}
			writeJSON(t, w, api.WireGuardSettings{Tunnels: &wgTunnels, Peers: &api.WGPeers{Item: &[]api.WGPeer{}}})
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := &tunnelResource{}
	r.client = client
	s := testResourceSchema(t, r)
	privateKey := s.Attributes["privatekey"].(schema.StringAttribute)
	if !privateKey.Sensitive {
		t.Fatal("tunnel private key must be sensitive")
	}
	addresses, diags := types.ListValueFrom(ctx, addressListType(), []ipAddressModel{
		{Address: types.StringValue("10.0.0.1/24"), Descr: types.StringValue("inside")},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	data := tunnelModel{
		Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
		PublicKey: types.StringValue("public"), Addresses: addresses,
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: testPlan(t, s, &data)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	var created tunnelModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatal(diags)
	}
	if created.PrivateKey.ValueString() != "private" || stored == nil ||
		stored.Addresses == nil || len(*stored.Addresses) != 1 {
		t.Fatalf("tunnel secret or addresses were not retained: state=%#v stored=%#v", created, stored)
	}

	data.Descr = types.StringValue("updated tunnel")
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, &data), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	var updated tunnelModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatal(diags)
	}
	if updated.PrivateKey.ValueString() != "private" || updated.Descr.ValueString() != "updated tunnel" {
		t.Fatalf("update lost tunnel fields: %#v", updated)
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() || stored != nil {
		t.Fatalf("tunnel delete failed: %v, stored=%#v", deleteResp.Diagnostics, stored)
	}
}

func TestPeerLifecycleRetainsMaskedPresharedKey(t *testing.T) {
	ctx := context.Background()
	var stored *api.WGPeer
	client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/vpn/wg/peer":
			var body api.WGPeer
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodPost && req.URL.Path == "/vpn/wg/peer/pub":
			var body api.WGPeer
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodDelete && req.URL.Path == "/vpn/wg/peer/pub":
			stored = nil
			writeJSON(t, w, api.PfsenseResult{Msg: "ok"})
		case req.Method == http.MethodGet && req.URL.Path == "/vpn/wg":
			peers := make([]api.WGPeer, 0)
			if stored != nil {
				masked := *stored
				masked.Presharedkey = nil
				peers = append(peers, masked)
			}
			writeJSON(t, w, api.WireGuardSettings{
				Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{}},
				Peers:   &api.WGPeers{Item: &peers},
			})
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	r := &peerResource{}
	r.client = client
	s := testResourceSchema(t, r)
	presharedKey := s.Attributes["presharedkey"].(schema.StringAttribute)
	if !presharedKey.Sensitive {
		t.Fatal("peer preshared key must be sensitive")
	}
	allowedIPs, diags := types.ListValueFrom(ctx, addressListType(), []ipAddressModel{
		{Address: types.StringValue("192.0.2.0/24"), Descr: types.StringNull()},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	data := peerModel{
		PublicKey: types.StringValue("pub"), PresharedKey: types.StringValue("secret"),
		AllowedIPs: allowedIPs,
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: testPlan(t, s, &data)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	var created peerModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatal(diags)
	}
	if created.PresharedKey.ValueString() != "secret" || stored == nil ||
		stored.Allowedips == nil || len(*stored.Allowedips) != 1 {
		t.Fatalf("peer secret or allowed IPs were not retained: state=%#v stored=%#v", created, stored)
	}

	data.Descr = types.StringValue("updated peer")
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, &data), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	var updated peerModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatal(diags)
	}
	if updated.PresharedKey.ValueString() != "secret" || updated.Descr.ValueString() != "updated peer" {
		t.Fatalf("update lost peer fields: %#v", updated)
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() || stored != nil {
		t.Fatalf("peer delete failed: %v, stored=%#v", deleteResp.Diagnostics, stored)
	}
}

func TestDeleteTreatsConfirmedAbsenceAsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		deletePath string
		newState   func(context.Context, *tfsdk.State) diag.Diagnostics
		delete     func(*testing.T, *api.ClientWithResponses, tfsdk.State)
	}{
		{
			name:       "tunnel",
			deletePath: "/vpn/wg/tunnel/wg0",
			newState: func(ctx context.Context, state *tfsdk.State) diag.Diagnostics {
				return state.Set(ctx, &tunnelModel{
					Name: types.StringValue("wg0"), PrivateKey: types.StringValue("secret"),
					PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
				})
			},
			delete: func(t *testing.T, client *api.ClientWithResponses, state tfsdk.State) {
				r := &tunnelResource{resourceBase{client: client}}
				resp := resource.DeleteResponse{State: state}
				r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
			},
		},
		{
			name:       "peer",
			deletePath: "/vpn/wg/peer/pub",
			newState: func(ctx context.Context, state *tfsdk.State) diag.Diagnostics {
				return state.Set(ctx, &peerModel{
					PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType()),
				})
			},
			delete: func(t *testing.T, client *api.ClientWithResponses, state tfsdk.State) {
				r := &peerResource{resourceBase{client: client}}
				resp := resource.DeleteResponse{State: state}
				r.Delete(context.Background(), resource.DeleteRequest{State: state}, &resp)
				if resp.Diagnostics.HasError() {
					t.Fatal(resp.Diagnostics)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
				switch {
				case req.Method == http.MethodDelete && req.URL.Path == tt.deletePath:
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"errmsg":"not found"}`))
				case req.Method == http.MethodGet && req.URL.Path == "/vpn/wg":
					writeJSON(t, w, api.WireGuardSettings{
						Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{}},
						Peers:   &api.WGPeers{Item: &[]api.WGPeer{}},
					})
				default:
					t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			})
			state := tfsdk.State{Schema: testResourceSchema(t, func() resource.Resource {
				if tt.name == "tunnel" {
					return &tunnelResource{}
				}
				return &peerResource{}
			}())}
			if diags := tt.newState(context.Background(), &state); diags.HasError() {
				t.Fatal(diags)
			}
			tt.delete(t, client, state)
		})
	}
}

func TestResourceReadRemovesOutOfBandDeletion(t *testing.T) {
	ctx := context.Background()
	client := testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeJSON(t, w, api.WireGuardSettings{
			Tunnels: &api.WGTunnels{Item: &[]api.WGTunnel{}},
			Peers:   &api.WGPeers{Item: &[]api.WGPeer{}},
		})
	})
	r := &tunnelResource{}
	r.client = client
	s := testResourceSchema(t, r)
	state := tfsdk.State{Schema: s}
	if diags := state.Set(ctx, &tunnelModel{
		Name: types.StringValue("wg0"), PrivateKey: types.StringValue("secret"),
		PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
	}); diags.HasError() {
		t.Fatal(diags)
	}
	resp := resource.ReadResponse{State: tfsdk.State{Schema: s}}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Fatalf("out-of-band tunnel deletion was not handled: %v", resp.Diagnostics)
	}
}

func TestValidateWriteResultRejectsInvalidResponse(t *testing.T) {
	tests := []struct {
		status int
		body   *api.PfsenseResult
	}{
		{status: http.StatusBadRequest, body: &api.PfsenseResult{Msg: "failed"}},
		{status: http.StatusOK},
		{status: http.StatusOK, body: &api.PfsenseResult{}},
	}
	for _, tt := range tests {
		if err := validateWriteResult(tt.status, tt.body, "test"); err == nil {
			t.Errorf("validateWriteResult(%d, %#v) unexpectedly succeeded", tt.status, tt.body)
		}
	}
	if err := validateWriteResult(http.StatusOK, &api.PfsenseResult{Msg: "ok"}, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestAddressListValueRejectsMissingAddress(t *testing.T) {
	empty := []api.WGIPAddress{{}}
	if _, err := addressListValue(context.Background(), &empty); err == nil {
		t.Fatal("expected an error for an address without a value")
	}
}

func TestGetTunnelConfigurationPropagatesAPIError(t *testing.T) {
	r := &tunnelResource{resourceBase{client: testAPIClient(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/vpn/wg" {
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	})}}
	if _, _, err := r.getTunnel(context.Background(), "wg0"); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("WireGuard API error was not propagated: %v", err)
	}
}
