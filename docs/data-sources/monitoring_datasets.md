# 🧭 pfsense_monitoring_datasets (Data Source)

Reads the available system monitoring datasets.

## 🧪 Example Usage

```terraform
data "pfsense_monitoring_datasets" "available" {}

output "monitoring_datasets" {
  value = jsondecode(data.pfsense_monitoring_datasets.available.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
