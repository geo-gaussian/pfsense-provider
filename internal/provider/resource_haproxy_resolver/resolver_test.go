package resource_haproxy_resolver

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

func resolverSchema(t *testing.T, r *resolverResource) schema.Schema {
	t.Helper()
	var response resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &response)
	return response.Schema
}

func resolverPlan(t *testing.T, s schema.Schema, value resolverModel) tfsdk.Plan {
	t.Helper()
	plan := tfsdk.Plan{Schema: s}
	if diags := plan.Set(context.Background(), value); diags.HasError() {
		t.Fatal(diags)
	}
	return plan
}

func TestResolverLifecycle(t *testing.T) {
	ctx := context.Background()
	var current *api.HaproxyResolver
	applies := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/ports/haproxy/resolvers":
			current = &api.HaproxyResolver{}
			if err := json.NewDecoder(req.Body).Decode(current); err != nil {
				t.Fatal(err)
			}
			id := "resolver-1"
			current.Id = &id
			_ = json.NewEncoder(w).Encode(current)
		case req.Method == http.MethodPost && req.URL.Path == "/ports/haproxy/resolvers/resolver-1":
			id := "resolver-1"
			current = &api.HaproxyResolver{Id: &id}
			if err := json.NewDecoder(req.Body).Decode(current); err != nil {
				t.Fatal(err)
			}
			_ = json.NewEncoder(w).Encode(current)
		case req.Method == http.MethodDelete && req.URL.Path == "/ports/haproxy/resolvers/resolver-1":
			current = nil
			_, _ = w.Write([]byte(`{"result":"success"}`))
		case req.Method == http.MethodPost && req.URL.Path == "/ports/haproxy/apply":
			applies++
			_, _ = w.Write([]byte(`{"result":"success"}`))
		case req.Method == http.MethodGet && req.URL.Path == "/ports/haproxy/resolvers":
			items := []api.HaproxyResolver{}
			if current != nil {
				items = append(items, *current)
			}
			_ = json.NewEncoder(w).Encode(api.HaproxyResolverList{Resolvers: &items})
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
	r := &resolverResource{client: client}
	s := resolverSchema(t, r)
	data := resolverModel{ID: types.StringUnknown(), Name: types.StringValue("first"),
		Server: types.StringNull(), Port: types.StringValue("53")}
	create := resource.CreateResponse{State: tfsdk.State{Schema: s}}
	r.Create(ctx, resource.CreateRequest{Plan: resolverPlan(t, s, data)}, &create)
	if create.Diagnostics.HasError() {
		t.Fatal(create.Diagnostics)
	}
	var created resolverModel
	if diags := create.State.Get(ctx, &created); diags.HasError() {
		t.Fatal(diags)
	}
	if created.ID.ValueString() != "resolver-1" || created.Name.ValueString() != "first" {
		t.Fatalf("unexpected created state: %#v", created)
	}
	created.Name = types.StringValue("second")
	update := resource.UpdateResponse{State: tfsdk.State{Schema: s}}
	r.Update(ctx, resource.UpdateRequest{Plan: resolverPlan(t, s, created), State: create.State}, &update)
	if update.Diagnostics.HasError() {
		t.Fatal(update.Diagnostics)
	}
	var updated resolverModel
	if diags := update.State.Get(ctx, &updated); diags.HasError() || updated.Name.ValueString() != "second" {
		t.Fatalf("unexpected updated state: %#v, %v", updated, diags)
	}
	read := resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() || read.State.Raw.IsNull() {
		t.Fatalf("unexpected read result: %v", read.Diagnostics)
	}
	deleted := resource.DeleteResponse{State: update.State}
	r.Delete(ctx, resource.DeleteRequest{State: update.State}, &deleted)
	if deleted.Diagnostics.HasError() || applies != 3 {
		t.Fatalf("delete diagnostics %v, apply count %d", deleted.Diagnostics, applies)
	}
	read = resource.ReadResponse{State: update.State}
	r.Read(ctx, resource.ReadRequest{State: update.State}, &read)
	if read.Diagnostics.HasError() || !read.State.Raw.IsNull() {
		t.Fatalf("out-of-band deletion was not detected: %v", read.Diagnostics)
	}
}

func TestResolverRejectsInvalidReadsAndApply(t *testing.T) {
	tests := []struct {
		name string
		body string
		path string
	}{
		{name: "missing list", path: "/ports/haproxy/resolvers", body: `{}`},
		{name: "invalid item", path: "/ports/haproxy/resolvers", body: `{"resolvers":[{}]}`},
		{name: "duplicate IDs", path: "/ports/haproxy/resolvers", body: `{"resolvers":[{"id":"x"},{"id":"x"}]}`},
		{name: "missing name", path: "/ports/haproxy/resolvers", body: `{"resolvers":[{"id":"x"}]}`},
		{name: "apply missing result", path: "/ports/haproxy/apply", body: `{}`},
		{name: "apply validation error", path: "/ports/haproxy/apply", body: `{"result":"failed","errors":"invalid"}`},
		{name: "apply failed result", path: "/ports/haproxy/apply", body: `{"result":"failed"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.URL.Path != tt.path {
					t.Errorf("unexpected path %s", req.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &resolverResource{client: client}
			if tt.path == "/ports/haproxy/apply" {
				err = r.apply(context.Background())
			} else {
				_, _, err = r.get(context.Background(), "x")
			}
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestResolverMutationFailures(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		status      int
		body        string
		applyStatus int
	}{
		{name: "create API failure", method: "create", status: http.StatusBadRequest, body: `{}`},
		{name: "create missing ID", method: "create", status: http.StatusOK, body: `{}`},
		{
			name: "create apply failure", method: "create", status: http.StatusOK,
			body: `{"id":"x"}`, applyStatus: http.StatusBadRequest,
		},
		{name: "update API failure", method: "update", status: http.StatusBadRequest, body: `{}`},
		{
			name: "update apply failure", method: "update", status: http.StatusOK,
			body: `{"id":"x"}`, applyStatus: http.StatusBadRequest,
		},
		{name: "delete API failure", method: "delete", status: http.StatusBadRequest, body: `{}`},
		{name: "delete invalid result", method: "delete", status: http.StatusOK, body: `{}`},
		{name: "delete failed result", method: "delete", status: http.StatusOK, body: `{"result":"failed"}`},
		{
			name: "delete apply failure", method: "delete", status: http.StatusOK,
			body: `{"result":"success"}`, applyStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if req.URL.Path == "/ports/haproxy/apply" {
					w.WriteHeader(tt.applyStatus)
					_, _ = w.Write([]byte(`{}`))
					return
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			r := &resolverResource{client: client}
			s := resolverSchema(t, r)
			data := resolverModel{ID: types.StringValue("x"), Name: types.StringValue("name"),
				Server: types.StringNull(), Port: types.StringNull()}
			state := tfsdk.State{Schema: s}
			if diags := state.Set(context.Background(), data); diags.HasError() {
				t.Fatal(diags)
			}
			switch tt.method {
			case "create":
				response := resource.CreateResponse{State: tfsdk.State{Schema: s}}
				r.Create(context.Background(), resource.CreateRequest{Plan: resolverPlan(t, s, data)}, &response)
				if !response.Diagnostics.HasError() {
					t.Fatal("expected create error")
				}
			case "update":
				response := resource.UpdateResponse{State: state}
				r.Update(context.Background(), resource.UpdateRequest{Plan: resolverPlan(t, s, data), State: state}, &response)
				if !response.Diagnostics.HasError() {
					t.Fatal("expected update error")
				}
			case "delete":
				response := resource.DeleteResponse{State: state}
				r.Delete(context.Background(), resource.DeleteRequest{State: state}, &response)
				if !response.Diagnostics.HasError() {
					t.Fatal("expected delete error")
				}
			}
		})
	}
}
