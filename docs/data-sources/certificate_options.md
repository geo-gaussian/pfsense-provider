# 🧭 pfsense_certificate_options (Data Source)

Reads the available certificate creation algorithms and key sizes.

## 🧪 Example Usage

```terraform
data "pfsense_certificate_options" "available" {}

output "certificate_options" {
  value = jsondecode(data.pfsense_certificate_options.available.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
