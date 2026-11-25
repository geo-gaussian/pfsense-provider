# 🗝️ pfsense_wireguard_key_pair (Ephemeral Resource)

Generates a WireGuard key pair using the firewall API. This is an action, so it
is exposed as an ephemeral resource rather than a repeatable data source.

Terraform 1.10 or later is required. Private key values are sensitive and are
available only during Terraform operations; they are not persisted in
Terraform state.

## 🧪 Example Usage

```terraform
ephemeral "pfsense_wireguard_key_pair" "site" {}
```

## 📋 Attribute Reference

- `private_key` (Computed, Sensitive) Generated private key.
- `private_key_clamped` (Computed, Sensitive) Generated clamped private key,
  when returned by the API.
- `public_key` (Computed) Generated public key.
- `was_clamped` (Computed) Whether the API clamped the private key.

The key pair is newly generated each time Terraform opens the ephemeral
resource. Ephemeral values are not persisted by this resource. A private key
copied into a stateful resource or output may still be stored in Terraform
state; use an ephemeral-compatible destination when they must remain transient.
