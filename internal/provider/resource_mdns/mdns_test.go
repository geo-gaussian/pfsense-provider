package resource_mdns

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

func testSchema(t *testing.T, r *mdnsResource) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	return response.Schema
}

func testPlan(t *testing.T, schema schema.Schema, model mdnsModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), model); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func TestMDNSLifecycle(t *testing.T) {
	ctx := context.Background()
	enabled := false
	reflection := true
	stored := api.MDNSConfig{Enable: &enabled, Reflection: &reflection}
	present := true
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/services/mdns" {
			t.Errorf("unexpected path %s", req.URL.Path)
		}
		if !present {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if req.Method == http.MethodPost {
			writes++
			if err := json.NewDecoder(req.Body).Decode(&stored); err != nil {
				t.Fatal(err)
			}
		} else if req.Method != http.MethodGet {
			t.Errorf("unexpected method %s", req.Method)
		}
		_ = json.NewEncoder(w).Encode(stored)
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := &mdnsResource{client: client}
	s := testSchema(t, r)
	data := mdnsModel{
		ID: types.StringUnknown(), Enable: types.BoolValue(true), CarpVIPInterface: types.StringUnknown(),
		IncludeInterfaces: types.ListUnknown(types.StringType), ExcludeInterfaces: types.ListUnknown(types.StringType),
		DisableIPv4: types.BoolUnknown(), DisableIPv6: types.BoolUnknown(), Reflection: types.BoolUnknown(),
		Publish: types.ObjectUnknown(publishTypes()),
	}
	create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: testPlan(t, s, data)}, &create)
	if create.Diagnostics.HasError() {
		t.Fatal(create.Diagnostics)
	}
	var created mdnsModel
	if diags := create.State.Get(ctx, &created); diags.HasError() {
		t.Fatal(diags)
	}
	if created.ID.ValueString() != "mdns" || !created.Enable.ValueBool() || !created.Reflection.ValueBool() {
		t.Fatalf("unexpected state: %#v", created)
	}
	created.Reflection = types.BoolValue(false)
	update := resource.UpdateResponse{State: create.State}
	r.Update(ctx, resource.UpdateRequest{Plan: testPlan(t, s, created), State: create.State}, &update)
	if update.Diagnostics.HasError() || stored.Reflection == nil || *stored.Reflection {
		t.Fatalf("update failed: %v", update.Diagnostics)
	}
	deleted := resource.DeleteResponse{State: update.State}
	r.Delete(ctx, resource.DeleteRequest{State: update.State}, &deleted)
	if deleted.Diagnostics.HasError() || stored.Enable == nil || *stored.Enable ||
		stored.Reflection == nil || *stored.Reflection || writes != 3 {
		t.Fatalf("delete failed: %v, config %#v, writes %d", deleted.Diagnostics, stored, writes)
	}
	present = false
	read := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("expected out-of-band deletion: %v", read.Diagnostics)
	}
}

func TestMDNSNestedSettingsRoundTrip(t *testing.T) {
	ctx := context.Background()
	lans, diags := types.ListValueFrom(ctx, types.StringType, []string{"lan"})
	if diags.HasError() {
		t.Fatal(diags)
	}
	publish, diags := types.ObjectValueFrom(ctx, publishTypes(), publishModel{
		Enable: types.BoolValue(true), Addresses: types.BoolValue(true), HostInfo: types.BoolNull(),
		Workstation: types.BoolNull(), Domain: types.BoolNull(), CustomHostname: types.StringValue("node"),
		CustomDomain: types.StringNull(), AAAAOverIPv4: types.BoolNull(), AOverIPv6: types.BoolNull(),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	data := mdnsModel{Enable: types.BoolValue(true), IncludeInterfaces: lans, Publish: publish}
	config := api.MDNSConfig{}
	if diags := data.merge(ctx, &config); diags.HasError() {
		t.Fatal(diags)
	}
	if config.IncludeInterfaces == nil || len(*config.IncludeInterfaces) != 1 ||
		config.Publish == nil || config.Publish.CustomHostname == nil || *config.Publish.CustomHostname != "node" {
		t.Fatalf("unexpected config %#v", config)
	}
	var restored mdnsModel
	if diags := restored.fromAPI(ctx, &config); diags.HasError() {
		t.Fatal(diags)
	}
	if !restored.Publish.Equal(publish) || !restored.IncludeInterfaces.Equal(lans) {
		t.Fatalf("unexpected restored state %#v", restored)
	}
}

func TestMDNSRejectsMalformedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`null`))
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := &mdnsResource{client: client}
	if _, _, err := r.get(context.Background()); err == nil {
		t.Fatal("expected invalid response error")
	}
	var imported resource.ImportStateResponse
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "wrong"}, &imported)
	if !imported.Diagnostics.HasError() {
		t.Fatal("expected invalid import ID error")
	}
}

func TestMDNSWriteAndDisableFailures(t *testing.T) {
	tests := []struct {
		name        string
		getStatus   int
		getBody     string
		postStatus  int
		postBody    string
		checkDelete bool
	}{
		{name: "GET fails", getStatus: http.StatusBadRequest, getBody: `{}`},
		{name: "settings missing", getStatus: http.StatusNotFound, getBody: `{}`},
		{
			name: "POST fails", getStatus: http.StatusOK, getBody: `{"enable":true}`,
			postStatus: http.StatusBadRequest, postBody: `{}`,
		},
		{
			name: "POST malformed", getStatus: http.StatusOK, getBody: `{"enable":true}`,
			postStatus: http.StatusOK, postBody: `null`,
		},
		{
			name: "disable not confirmed", getStatus: http.StatusOK, getBody: `{"enable":true}`,
			postStatus: http.StatusOK, postBody: `{"enable":true}`, checkDelete: true,
		},
		{
			name: "disable verification fails", getStatus: http.StatusOK, getBody: `{"enable":true}`,
			postStatus: http.StatusOK, postBody: `{"enable":false}`, checkDelete: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.Method == http.MethodGet {
					w.WriteHeader(tt.getStatus)
					_, _ = w.Write([]byte(tt.getBody))
					return
				}
				w.WriteHeader(tt.postStatus)
				_, _ = w.Write([]byte(tt.postBody))
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &mdnsResource{client: client}
			if tt.checkDelete {
				var response resource.DeleteResponse
				r.Delete(context.Background(), resource.DeleteRequest{}, &response)
				if !response.Diagnostics.HasError() {
					t.Fatal("expected disable error")
				}
				return
			}
			data := mdnsModel{Enable: types.BoolValue(true)}
			if _, err := r.write(context.Background(), &data); err == nil {
				t.Fatal("expected write error")
			}
		})
	}
}
