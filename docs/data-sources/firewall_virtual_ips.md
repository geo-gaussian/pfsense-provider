# 🧭 pfsense_firewall_virtual_ips (Data Source)

Reads configured firewall virtual IPs.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_virtual_ips" "current" {}

output "virtual_ips" {
  value = jsondecode(data.pfsense_firewall_virtual_ips.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API, including virtual IPs and
  available interfaces.

This data source does not apply configuration or reload the firewall. The
response has only been tested against mock servers; appliance behaviour and
compatibility remain unverified.
