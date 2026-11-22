# 🧭 pfsense_firewall_altq (Data Source)

Reads configured ALTQ queues.

## 🧪 Example Usage

```terraform
data "pfsense_firewall_altq" "current" {}

output "altq" {
  value = jsondecode(data.pfsense_firewall_altq.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
