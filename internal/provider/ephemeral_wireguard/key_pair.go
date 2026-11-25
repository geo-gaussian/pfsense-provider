package ephemeral_wireguard

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ ephemeral.EphemeralResource = &keyPairResource{}
var _ ephemeral.EphemeralResourceWithConfigure = &keyPairResource{}

type keyPairResource struct {
	client *api.ClientWithResponses
}

type keyPairModel struct {
	PrivateKey        types.String `tfsdk:"private_key"`
	PrivateKeyClamped types.String `tfsdk:"private_key_clamped"`
	PublicKey         types.String `tfsdk:"public_key"`
	WasClamped        types.Bool   `tfsdk:"was_clamped"`
}

func New() ephemeral.EphemeralResource {
	return &keyPairResource{}
}

func (r *keyPairResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wireguard_key_pair"
}

func (r *keyPairResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates a WireGuard key pair on the configured firewall. Terraform 1.10 or later is required. Private key outputs are ephemeral and sensitive.",
		Attributes: map[string]schema.Attribute{
			"private_key":         schema.StringAttribute{Computed: true, Sensitive: true},
			"private_key_clamped": schema.StringAttribute{Computed: true, Sensitive: true},
			"public_key":          schema.StringAttribute{Computed: true},
			"was_clamped":         schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *keyPairResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*api.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider configuration", "The WireGuard key-pair ephemeral resource received an unexpected API client.")
	}
}

func (r *keyPairResource) Open(ctx context.Context, _ ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.client == nil {
		resp.Diagnostics.AddError("Unable to generate WireGuard keys", "The API client is not configured.")
		return
	}
	result, err := r.client.GetNewWireGuardKeysWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to generate WireGuard keys", err.Error())
		return
	}
	if result.StatusCode() != http.StatusOK || result.JSON200 == nil {
		resp.Diagnostics.AddError("Unable to generate WireGuard keys", fmt.Sprintf("API returned HTTP %d or an invalid JSON response.", result.StatusCode()))
		return
	}
	if result.JSON200.Privkey == nil || len(*result.JSON200.Privkey) == 0 ||
		result.JSON200.Pubkey == nil || len(*result.JSON200.Pubkey) == 0 {
		resp.Diagnostics.AddError("Unable to generate WireGuard keys", "The API response did not include both private and public keys.")
		return
	}

	data := keyPairModel{
		PrivateKey: types.StringValue(base64.StdEncoding.EncodeToString(*result.JSON200.Privkey)),
		PublicKey:  types.StringValue(base64.StdEncoding.EncodeToString(*result.JSON200.Pubkey)),
	}
	if result.JSON200.PrivkeyClamped != nil {
		data.PrivateKeyClamped = types.StringValue(base64.StdEncoding.EncodeToString(*result.JSON200.PrivkeyClamped))
	} else {
		data.PrivateKeyClamped = types.StringNull()
	}
	if result.JSON200.WasClamped != nil {
		data.WasClamped = types.BoolValue(*result.JSON200.WasClamped)
	} else {
		data.WasClamped = types.BoolNull()
	}
	resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
}
