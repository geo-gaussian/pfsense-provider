# 🧭 pfsense_firewall_traffic_shaper (Data Source)

Reads configured firewall traffic shapers.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_traffic_shaper" "current" {}

output "traffic_shaper" {
  value = jsondecode(data.pfsense_firewall_traffic_shaper.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
