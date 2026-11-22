# 🧭 pfsense_gif_interfaces (Data Source)

Reads configured GIF tunnel interfaces.

## 🧪 Example Usage

```terraform
data "pfsense_gif_interfaces" "current" {}

output "gif_interfaces" {
  value = jsondecode(data.pfsense_gif_interfaces.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
