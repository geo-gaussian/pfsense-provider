package resource_firewall_alias

import (
	"context"
	"strings"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestFromAPIReportsFactoryFailures(t *testing.T) {
	targets := []api.FWTarget{{Name: "example.test"}}
	value := &api.FWAlias{Name: "example", Targets: &targets}
	failure := diag.Diagnostics{diag.NewErrorDiagnostic("factory failure", "invalid value")}
	tests := []struct {
		name      string
		newTarget func(map[string]attr.Type, map[string]attr.Value) (TargetsValue, diag.Diagnostics)
		newList   func(attr.Type, []attr.Value) (basetypes.ListValue, diag.Diagnostics)
		want      string
	}{
		{
			name: "target",
			newTarget: func(map[string]attr.Type, map[string]attr.Value) (TargetsValue, diag.Diagnostics) {
				return TargetsValue{}, failure
			},
			newList: types.ListValue,
			want:    "invalid target returned by API",
		},
		{
			name:      "target list",
			newTarget: NewTargetsValue,
			newList: func(attr.Type, []attr.Value) (basetypes.ListValue, diag.Diagnostics) {
				return basetypes.ListValue{}, failure
			},
			want: "invalid target list returned by API",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var model FirewallAliasModel
			err := model.fromAPIWithFactories(context.Background(), value, tt.newTarget, tt.newList)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), "invalid value") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
