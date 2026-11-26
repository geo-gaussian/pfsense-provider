package resource_firewall_alias

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestAliasLifecycle(t *testing.T) {
	ctx := context.Background()
	var stored *api.FWAliasReq
	var reloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.URL.Path == "/aliases" && req.Method == http.MethodPut,
			req.URL.Path == "/aliases/example" && req.Method == http.MethodPost:
			var body api.FWAliasReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/aliases/example" && req.Method == http.MethodGet:
			if stored == nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"errmsg":"alias not found"}`)
				return
			}
			if err := json.NewEncoder(w).Encode(stored); err != nil {
				t.Error(err)
			}
		case req.URL.Path == "/aliases" && req.Method == http.MethodGet:
			_, _ = fmt.Fprint(w, `{"aliases":[]}`)
		case req.URL.Path == "/aliases/example" && req.Method == http.MethodDelete:
			stored = nil
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/system/filterreload/reload" && req.Method == http.MethodPost:
			reloads++
			_, _ = fmt.Fprint(w, `{"result":"started"}`)
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	r := &aliasResource{client: client}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	model := aliasTestModel(t, ctx)
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	createResp := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	if stored == nil || stored.Targets == nil || len(*stored.Targets) != 1 || (*stored.Targets)[0].Name != "example.test" {
		t.Fatalf("unexpected stored alias: %#v", stored)
	}

	model.Descr = types.StringValue("updated")
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if stored.Descr == nil || *stored.Descr != "updated" {
		t.Fatalf("description was not updated: %#v", stored)
	}

	deleteResp := resource.DeleteResponse{State: updateResp.State}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatal(deleteResp.Diagnostics)
	}
	if stored != nil || reloads != 3 {
		t.Fatalf("stored = %#v, reloads = %d", stored, reloads)
	}
	readResp := resource.ReadResponse{State: updateResp.State}
	r.Read(ctx, resource.ReadRequest{State: updateResp.State}, &readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Fatalf("out-of-band deletion was not removed: %v", readResp.Diagnostics)
	}
}

func aliasTestModel(t *testing.T, ctx context.Context) FirewallAliasModel {
	t.Helper()
	targetType := TargetsType{ObjectType: types.ObjectType{AttrTypes: TargetsValue{}.AttributeTypes(ctx)}}
	target, diags := NewTargetsValue(TargetsValue{}.AttributeTypes(ctx), map[string]attr.Value{
		"name": types.StringValue("example.test"), "descr": types.StringNull(), "updatefreq": types.StringNull(),
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	targets, diags := types.ListValue(targetType, []attr.Value{target})
	if diags.HasError() {
		t.Fatal(diags)
	}
	return FirewallAliasModel{
		Name: types.StringValue("example"), Type: types.StringValue("host"),
		Address: types.StringNull(), Descr: types.StringNull(), Detail: types.StringNull(),
		Tags: types.StringNull(), Updatefreq: types.StringNull(), Targets: targets,
	}
}

func TestReadRejectsTruncatedAlias(t *testing.T) {
	value := api.FWAlias{Name: "example"}
	truncated := true
	value.Truncated = &truncated
	var model FirewallAliasModel
	if err := model.fromAPI(context.Background(), &value); err == nil {
		t.Fatal("expected an error for a truncated response")
	}
}
func TestBadRequestDoesNotRemoveExistingAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/aliases/example" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"errmsg":"read failed"}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"aliases":[{"name":"example"}]}`)
	}))
	defer server.Close()
	client, err := api.NewClientWithResponses(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, missing, err := (&aliasResource{client: client}).get(context.Background(), "example")
	if err == nil || missing {
		t.Fatalf("existing alias was treated as absent: missing=%v, err=%v", missing, err)
	}
}
