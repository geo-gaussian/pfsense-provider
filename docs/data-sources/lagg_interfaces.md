# 🧭 pfsense_lagg_interfaces (Data Source)

Reads configured link aggregation interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_lagg_interfaces" "current" {}

output "lagg_interfaces" {
  value = jsondecode(data.pfsense_lagg_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
