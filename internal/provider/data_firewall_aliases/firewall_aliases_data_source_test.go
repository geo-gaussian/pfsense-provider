package data_firewall_aliases

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geo-gaussian/pfsense-provider/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

func TestReadAliases(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantErr  bool
		wantNull bool
		count    int
	}{
		{name: "populated", status: 200, body: `{"aliases":[{"name":"example","type":"host","descr":"test","truncated":true}]}`, count: 1},
		{name: "empty", status: 200, body: `{"aliases":[]}`},
		{name: "missing list", status: 200, body: `{}`, wantNull: true},
		{name: "missing name", status: 200, body: `{"aliases":[{}]}`, wantErr: true},
		{name: "bad response", status: 400, body: `{"errmsg":"failure"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet || req.URL.Path != "/aliases" {
					t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			client, err := api.NewClientWithResponses(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			d := &aliasesDataSource{client: client}
			var schemaResp datasource.SchemaResponse
			d.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResp)
			resp := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			d.Read(context.Background(), datasource.ReadRequest{}, &resp)
			if resp.Diagnostics.HasError() != tt.wantErr {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if tt.wantErr {
				return
			}
			var data aliasesModel
			if diags := resp.State.Get(context.Background(), &data); diags.HasError() {
				t.Fatal(diags)
			}
			if len(data.Aliases) != tt.count {
				t.Fatalf("got %d aliases, want %d", len(data.Aliases), tt.count)
			}
			if (data.Aliases == nil) != tt.wantNull {
				t.Fatalf("aliases null = %v, want %v", data.Aliases == nil, tt.wantNull)
			}
			if tt.wantNull && len(resp.Diagnostics.Warnings()) == 0 {
				t.Fatal("expected a warning for the omitted alias list")
			}
			if tt.count == 1 && (data.Aliases[0].Name.ValueString() != "example" ||
				data.Aliases[0].Type.ValueString() != "host" ||
				data.Aliases[0].Descr.ValueString() != "test" ||
				!data.Aliases[0].Truncated.ValueBool()) {
				t.Fatalf("unexpected alias: %#v", data.Aliases[0])
			}
		})
	}
}
