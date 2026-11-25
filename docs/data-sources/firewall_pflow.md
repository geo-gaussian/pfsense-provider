# 🧭 pfsense_firewall_pflow (Data Source)

Reads global packet-flow settings and exporters.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_pflow" "current" {}

output "pflow" {
  value = jsondecode(data.pfsense_firewall_pflow.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API, including options, exporters,
  and available source addresses.

This data source does not apply configuration or reload the firewall. The
response has only been tested against mock servers; appliance behaviour and
compatibility remain unverified.
