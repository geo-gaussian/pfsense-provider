# 🧭 pfsense_gre_interfaces (Data Source)

Reads configured GRE tunnel interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_gre_interfaces" "current" {}

output "gre_interfaces" {
  value = jsondecode(data.pfsense_gre_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
