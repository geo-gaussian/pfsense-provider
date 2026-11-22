# 🧭 pfsense_firewall_nat_npt (Data Source)

Reads configured IPv6 Network Prefix Translation rules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_nat_npt" "current" {}

output "npt_rules" {
  value = jsondecode(data.pfsense_firewall_nat_npt.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
