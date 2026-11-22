# 🧭 pfsense_queue_stats (Data Source)

Reads traffic-shaping queue statistics.

## 🧪 Example Usage

```terraform
data "pfsense_queue_stats" "current" {}

output "queue_stats" {
  value = jsondecode(data.pfsense_queue_stats.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
