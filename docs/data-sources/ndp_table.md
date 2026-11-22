# 🧭 pfsense_ndp_table (Data Source)

Reads the IPv6 neighbor discovery table, including local device addresses. The
JSON result is marked sensitive.

## 🧪 Example Usage

```terraform
data "pfsense_ndp_table" "current" {}
```

## 📋 Attribute Reference

- `json` — Complete sensitive JSON response from the API.

Use `jsondecode()` to access response fields. Terraform state and saved plans
contain local network information; protect them accordingly.
