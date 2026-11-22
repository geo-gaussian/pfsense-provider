# 🧭 pfsense_firewall_limiters (Data Source)

Reads configured firewall limiters.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_limiters" "current" {}

output "limiters" {
  value = jsondecode(data.pfsense_firewall_limiters.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
