# 🧭 pfsense_firewall_pflow (Data Source)

Reads global packet-flow settings and the configured exporters.

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

Manage individual exporters with the
[pfsense_firewall_pflow_exporter resource](../resources/firewall_pflow_exporter.md).
Global PFlow options remain read-only because the API does not provide a reset
operation. This data source does not apply configuration or reload the
firewall. The response has only been tested against mock servers; appliance
behaviour and compatibility remain unverified.
