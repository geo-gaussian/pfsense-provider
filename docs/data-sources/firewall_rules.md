# 🧭 pfsense_firewall_rules (Data Source)

Reads firewall rules and related lookup values for one interface.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_rules" "lan" {
  interface = "lan"
}

output "lan_rules" {
  value = jsondecode(data.pfsense_firewall_rules.lan.json)
}
```

## 📋 Attribute Reference

- `interface` (Required) — Firewall interface whose rules are read.
- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
