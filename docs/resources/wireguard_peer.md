# 🤝 pfsense_wireguard_peer (Resource)

Manages a WireGuard peer.

!> WireGuard operations have only been tested against mock servers. Appliance
behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_wireguard_peer" "site" {
  publickey           = var.peer_public_key
  tun                 = pfsense_wireguard_tunnel.site.name
  endpoint            = "peer.example.test"
  port                = "51820"
  persistentkeepalive = "25"
  presharedkey        = var.peer_preshared_key
  allowedips          = [{ address = "10.0.1.0/24" }]
}
```

## ⚙️ Argument Reference

- `publickey` (Required) Peer public key. Changing it replaces the resource.
- `enabled` (Optional) Whether the peer is enabled.
- `tun` (Optional) Associated tunnel name.
- `descr` (Optional) Peer description.
- `endpoint` (Optional) Peer endpoint.
- `port` (Optional) Peer UDP port.
- `persistentkeepalive` (Optional) Persistent keepalive interval.
- `presharedkey` (Optional, Sensitive) Peer preshared key. If the API masks it
  during reads, the provider retains its prior state value.
- `allowedips` (Optional) List of allowed IP ranges:
  - `address` (Required) CIDR address and mask.
  - `descr` (Optional) Address description.

## 📥 Import

Import an existing peer by public key:

```shell
terraform import pfsense_wireguard_peer.site '<peer-public-key>'
```
