# 🧭 pfsense_mdns (Data Source)

Reads multicast DNS service settings.

## 🧪 Example Usage

```terraform
data "pfsense_mdns" "current" {}

output "mdns" {
  value = jsondecode(data.pfsense_mdns.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
