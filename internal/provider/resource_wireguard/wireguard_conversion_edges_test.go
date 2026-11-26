package resource_wireguard

import (
	"context"

	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

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

func TestAddressListValueReportsFrameworkConversionErrors(t *testing.T) {
	_, err := addressListValueWithFactory(
		context.Background(),
		&[]api.WGIPAddress{{Address: "192.0.2.1/32"}},
		func(context.Context, attr.Type, any) (basetypes.ListValue, diag.Diagnostics) {
			return types.ListNull(addressListType()), diag.Diagnostics{
				diag.NewErrorDiagnostic("List conversion failed", "conversion rejected"),
			}
		},
	)
	if err == nil || err.Error() != "convert WireGuard IP addresses: conversion rejected" {
		t.Fatalf("unexpected address-list conversion error: %v", err)
	}
}

func TestWireGuardLifecycleReportsAddressConversionErrors(t *testing.T) {
	ctx := context.Background()
	for _, resourceName := range []string{"tunnel", "peer"} {
		for _, operation := range []string{"create", "update"} {
			t.Run(resourceName+" "+operation, func(t *testing.T) {
				var target resource.Resource
				var model any
				var stateModel any
				var listAttribute string
				switch resourceName {
				case "tunnel":
					target = NewTunnel()
					listAttribute = "addresses"
					model = &tunnelModel{
						Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
						PublicKey: types.StringValue("public"),
						Addresses: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid")}),
					}
					stateModel = &tunnelModel{
						Name: types.StringValue("wg0"), PrivateKey: types.StringValue("private"),
						PublicKey: types.StringValue("public"), Addresses: types.ListNull(addressListType()),
					}
				case "peer":
					target = NewPeer()
					listAttribute = "allowedips"
					model = &peerModel{
						PublicKey:  types.StringValue("pub"),
						AllowedIPs: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("invalid")}),
					}
					stateModel = &peerModel{
						PublicKey: types.StringValue("pub"), AllowedIPs: types.ListNull(addressListType()),
					}
				}

				planSchema := testResourceSchema(t, target)
				planSchema.Attributes[listAttribute] = schema.ListAttribute{
					Optional:    true,
					ElementType: types.StringType,
				}
				plan := tfsdk.Plan{Schema: planSchema}
				if diags := plan.Set(ctx, model); diags.HasError() {
					t.Fatal(diags)
				}

				if operation == "create" {
					resp := resource.CreateResponse{State: tfsdk.State{Schema: testResourceSchema(t, target)}}
					callCreate(target, ctx, plan, &resp)
					if !resp.Diagnostics.HasError() {
						t.Fatal("expected invalid list conversion to fail before the API request")
					}
					return
				}

				state := tfsdk.State{Schema: testResourceSchema(t, target)}
				if diags := state.Set(ctx, stateModel); diags.HasError() {
					t.Fatal(diags)
				}
				resp := resource.UpdateResponse{State: tfsdk.State{Schema: state.Schema}}
				callUpdate(target, ctx, plan, state, &resp)
				if !resp.Diagnostics.HasError() {
					t.Fatal("expected invalid list conversion to fail before the API request")
				}
			})
		}
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
