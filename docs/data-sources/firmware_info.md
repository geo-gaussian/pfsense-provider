# 🧭 pfsense_firmware_info (Data Source)

Reads the installed firmware version and available upgrade information. It
does not install or upgrade firmware.

## 🧪 Example Usage

```terraform
data "pfsense_firmware_info" "current" {}

output "firmware_info" {
  value = jsondecode(data.pfsense_firmware_info.current.json)
}
```

## 📋 Attribute Reference

- `json` — Complete JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain the response; protect them accordingly.
