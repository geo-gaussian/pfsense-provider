package resource_wireguard

import "github.com/hashicorp/terraform-plugin-framework/resource"

func NewSettings() resource.Resource {
	return &settingsResource{}
}

func NewTunnel() resource.Resource {
	return &tunnelResource{}
}

func NewPeer() resource.Resource {
	return &peerResource{}
}
