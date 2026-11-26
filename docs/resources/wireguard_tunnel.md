# 🔐 pfsense_wireguard_tunnel (Resource)

Manages a WireGuard tunnel.

!> WireGuard operations have only been tested against mock servers. Appliance
behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
variable "wireguard_private_key" {
  type      = string
  sensitive = true
}

variable "wireguard_public_key" {
  type = string
}

resource "pfsense_wireguard_tunnel" "site" {
  name       = "wg0"
  privatekey = var.wireguard_private_key
  publickey  = var.wireguard_public_key
  enabled    = true
  listenport = 51820
  addresses  = [{ address = "10.0.0.1/24" }]
}
```

## ⚙️ Argument Reference

- `name` (Required) Tunnel name. Changing it replaces the resource.
- `privatekey` (Optional, Sensitive) Tunnel private key. It must be provided
  when creating or updating a tunnel. If the API omits it or returns an empty
  value during reads, the provider retains its prior state value. Sensitive
  values can remain in Terraform state; protect state and saved plans.
- `publickey` (Required) Tunnel public key.
- `descr` (Optional) Tunnel description.
- `enabled` (Optional) Whether the tunnel is enabled.
- `listenport` (Optional) UDP listen port.
- `mtu` (Optional) Tunnel MTU.
- `addresses` (Optional) List of tunnel addresses:
  - `address` (Required) CIDR address and mask.
  - `descr` (Optional) Address description.

## 📥 Import

Import an existing tunnel by name. If the API omits its private key or returns
an empty value, supply the key in configuration before making changes that
require a tunnel write.

```shell
terraform import pfsense_wireguard_tunnel.site wg0
```
