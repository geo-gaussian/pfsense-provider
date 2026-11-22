# 🧭 pfsense_firewall_nat_port_forward (Data Source)

Reads configured firewall port-forward NAT rules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_nat_port_forward" "current" {}

output "nat_rules" {
  value = jsondecode(data.pfsense_firewall_nat_port_forward.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
