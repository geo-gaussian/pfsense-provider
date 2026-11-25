package data_readonly

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func NewIPSecWidget() datasource.DataSource {
	return newDataSource("ipsec_widget", "Reads IPsec tunnel and connection summary statistics.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetIPSecWidgetWithResponse(ctx)
		},
	)
}

func NewOpenVPNStatus() datasource.DataSource {
	return newDataSource("openvpn_status", "Reads active OpenVPN server and client status.", nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetOpenVPNStatusWithResponse(ctx)
		},
	)
}

func NewWireGuardConfig() datasource.DataSource {
	return newSensitiveDataSource(
		"wireguard_config",
		"Reads the full WireGuard configuration, including private-key fields. Protect Terraform state and saved plans.",
		nil,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			return client.GetWireGuardWithResponse(ctx)
		},
	)
}

func NewWireGuardTunnelConf() datasource.DataSource {
	return newSensitiveDataSource(
		"wireguard_tunnel_conf",
		"Reads a rendered WireGuard tunnel configuration. The result can contain private keys.",
		map[string]schema.Attribute{
			"name": schema.StringAttribute{Required: true, Description: "WireGuard tunnel name."},
		},
		func(ctx context.Context, client *api.ClientWithResponses, arguments map[string]string) (apiResponse, error) {
			return client.GetWireGuardTunnelConfWithResponse(ctx, arguments["name"])
		},
	)
}

func NewWireGuardStatus() datasource.DataSource {
	return newWireGuardStatusDataSource(json.Marshal)
}

func newWireGuardStatusDataSource(marshalJSON func(any) ([]byte, error)) datasource.DataSource {
	return newDataSourceWithJSONDescription(
		"wireguard_status",
		"Reads WireGuard operational status without returning private keys, preshared keys, or configuration objects.",
		nil,
		"A JSON projection of WireGuard status. Private keys, preshared keys, and configuration objects are omitted.",
		false,
		func(ctx context.Context, client *api.ClientWithResponses, _ map[string]string) (apiResponse, error) {
			result, err := client.GetWireGuardStatusWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if result.StatusCode() != 200 {
				return result, nil
			}
			if result.JSON200 == nil {
				return nil, errors.New("the API returned an invalid WireGuard status response")
			}

			safeStatus := projectWireGuardStatus(result.JSON200)
			body, err := marshalJSON(safeStatus)
			if err != nil {
				return nil, err
			}
			return jsonResponse{statusCode: result.StatusCode(), body: body}, nil
		},
	)
}

type jsonResponse struct {
	statusCode int
	body       []byte
}

func (r jsonResponse) StatusCode() int {
	return r.statusCode
}

func (r jsonResponse) GetBody() []byte {
	return r.body
}

type wireGuardStatusJSON struct {
	Status *[]wireGuardTunnelStatusJSON `json:"status,omitempty"`
}

type wireGuardTunnelStatusJSON struct {
	Fwmark     *string                    `json:"fwmark,omitempty"`
	Inpkts     *int                       `json:"inpkts,omitempty"`
	ListenPort *string                    `json:"listen_port,omitempty"`
	Mtu        *int                       `json:"mtu,omitempty"`
	Outpkts    *int                       `json:"outpkts,omitempty"`
	Peers      *[]wireGuardPeerStatusJSON `json:"peers,omitempty"`
	PublicKey  *string                    `json:"public_key,omitempty"`
	Status     *string                    `json:"status,omitempty"`
	TransferRx *int                       `json:"transfer_rx,omitempty"`
	TransferTx *int                       `json:"transfer_tx,omitempty"`
}

type wireGuardPeerStatusJSON struct {
	AllowedIPs          *string `json:"allowed_ips,omitempty"`
	Endpoint            *string `json:"endpoint,omitempty"`
	LatestHandshake     *string `json:"latest_handshake,omitempty"`
	PersistentKeepalive *string `json:"persistent_keepalive,omitempty"`
	PublicKey           *string `json:"public_key,omitempty"`
	TransferRx          *string `json:"transfer_rx,omitempty"`
	TransferTx          *string `json:"transfer_tx,omitempty"`
}

func projectWireGuardStatus(status *api.WireGuardStatus) wireGuardStatusJSON {
	projected := wireGuardStatusJSON{}
	if status.Status == nil {
		return projected
	}

	tunnels := make([]wireGuardTunnelStatusJSON, 0, len(*status.Status))
	for _, tunnel := range *status.Status {
		projectedTunnel := wireGuardTunnelStatusJSON{
			Fwmark:     tunnel.Fwmark,
			Inpkts:     tunnel.Inpkts,
			ListenPort: tunnel.ListenPort,
			Mtu:        tunnel.Mtu,
			Outpkts:    tunnel.Outpkts,
			PublicKey:  tunnel.PublicKey,
			Status:     tunnel.Status,
			TransferRx: tunnel.TransferRx,
			TransferTx: tunnel.TransferTx,
		}
		if tunnel.Peers != nil {
			peers := make([]wireGuardPeerStatusJSON, 0, len(*tunnel.Peers))
			for _, peer := range *tunnel.Peers {
				peers = append(peers, wireGuardPeerStatusJSON{
					AllowedIPs:          peer.AllowedIps,
					Endpoint:            peer.Endpoint,
					LatestHandshake:     peer.LatestHandshake,
					PersistentKeepalive: peer.PersistentKeepalive,
					PublicKey:           peer.PublicKey,
					TransferRx:          peer.TransferRx,
					TransferTx:          peer.TransferTx,
				})
			}
			projectedTunnel.Peers = &peers
		}
		tunnels = append(tunnels, projectedTunnel)
	}
	projected.Status = &tunnels
	return projected
}
