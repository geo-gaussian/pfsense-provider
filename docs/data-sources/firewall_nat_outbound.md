# 🧭 pfsense_firewall_nat_outbound (Data Source)

Reads configured outbound NAT rules.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_nat_outbound" "current" {}

output "nat_rules" {
  value = jsondecode(data.pfsense_firewall_nat_outbound.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.

The Nexus API does not provide an outbound-rule delete operation, so outbound
NAT remains read-only in this provider rather than being exposed as a Terraform
resource without a safe destroy lifecycle.
