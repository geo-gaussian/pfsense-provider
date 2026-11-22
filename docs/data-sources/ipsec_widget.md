# 🧭 pfsense_ipsec_widget (Data Source)

Reads IPsec tunnel and connection summary statistics.

## 🧪 Example Usage

```terraform
data "pfsense_ipsec_widget" "current" {}

output "ipsec_status" {
  value = jsondecode(data.pfsense_ipsec_widget.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
