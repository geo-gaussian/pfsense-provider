# 🔧 pfsense_haproxy_resolver (Resource)

Manages an HAProxy DNS resolver. Changes apply the complete HAProxy
configuration, including changes staged outside Terraform.

If applying fails after a write, inspect the HAProxy configuration and apply
it manually before retrying. Terraform refresh does not retry the apply.

!> This resource has only been tested against mock servers. Appliance
behaviour and compatibility remain unverified.

## 🧪 Example Usage

```terraform
resource "pfsense_haproxy_resolver" "example" {
  name = "example"
}
```

## ⚙️ Argument Reference

- `name` (Required) Unique resolver name.
- `server` (Optional) DNS server address.
- `port` (Optional) DNS server port.
- `id` (Computed) Identifier assigned by the API.

## 📥 Import

```shell
terraform import pfsense_haproxy_resolver.example RESOLVER_ID
```
