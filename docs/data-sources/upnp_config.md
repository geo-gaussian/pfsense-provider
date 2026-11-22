# 🧭 pfsense_upnp_config (Data Source)

Reads UPnP and NAT-PMP service settings and access rules.

## 🧪 Example Usage

```terraform
data "pfsense_upnp_config" "current" {}

output "upnp_config" {
  value = jsondecode(data.pfsense_upnp_config.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
