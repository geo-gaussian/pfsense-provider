# 🧭 pfsense_firewall_nat_one_to_one (Data Source)

Reads configured one-to-one NAT rules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_nat_one_to_one" "current" {}

output "nat_rules" {
  value = jsondecode(data.pfsense_firewall_nat_one_to_one.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
