# 🛡️ Terraform Provider for pfSense Nexus (`pfsense-nexus`)

A custom Terraform provider built on HashiCorp's `terraform-plugin-framework` and generated from Netgate's official OpenAPI specification (`nexus.openapi.yaml`) to manage **pfSense Plus (26.07+)** and **pfSense (26.10+)** via the native Go-based REST API.

## ✨ Key Features

- **Native Go API Integration:** Communicates directly with the built-in Nexus REST API (`:8443`) using Bearer JWT authentication—no HTML scraping or legacy PHP packages required.
- **OpenAPI-Driven Scaffolding:** Resource schemas and models are generated directly from Netgate's official `nexus.openapi.yaml` via `tfplugingen-openapi`.
- **Thread-Safe Router Commits:** Built-in `sync.Mutex` serializes API writes so parallel Terraform graph execution never triggers configuration lock collisions on pfSense.
- **Brownfield Code Generation:** Supports `import` blocks and `terraform plan -generate-config-out` to export existing firewall rules, aliases, VLANs, and WireGuard tunnels into HCL.

## 🛠️ Prerequisites

- **Go** `1.23+`
- **Terraform** `1.8+` (or OpenTofu `1.8+`)
- **pfSense** `26.07+` (supported Netgate/cloud hardware) or `26.10+` (any `amd64` hardware/VM) with the Nexus service enabled on port `8443`.
- HashiCorp code generators:
  ```sh
  go install ://github.com
  go install ://github.com
  ```

## 🚀 Generating & Building the Provider

1. **Fetch the latest Netgate OpenAPI specification:**

   ```sh
   curl -o docs/nexus.openapi.yaml https://githubusercontent.com
   ```

2. **Generate Terraform Plugin Framework Go code:**

   ```sh
   tfplugingen-openapi generate \
     --config generator_config.yml \
     --output provider_code_spec.json \
     docs/nexus.openapi.yaml

   tfplugingen-framework generate all \
     --input provider_code_spec.json \
     --output internal/provider
   ```

3. **Compile the binary:**
   ```sh
   go mod tidy
   go build -o pfsense-provider
   ```

## 💻 Local Development Setup (`~/.terraformrc`)

Configure Terraform to use your locally compiled binary without publishing to the public registry:

```hcl
provider_installation {
  dev_overrides {
    "local/pfsense-nexus" = "/absolute/path/to/pfsense-provider"
  }
  direct {}
}
```

_(Skip `terraform init` when `dev_overrides` is active and run `terraform plan` directly.)_

## 📝 Example Configuration

```hcl
terraform {
  required_providers {
    pfsense-nexus = {
      source = "local/pfsense-nexus"
    }
  }
}

provider "pfsense-nexus" {
  url      = "https://192.168.1.1:8443"
  username = "terraform-api"
  password = var.pfsense_password
  insecure = true
}

resource "pfsense_wireguard_tunnel" "wg0" {
  name        = "tun_wg0"
  description = "Site-to-Site WireGuard"
  listen_port = 51820
  private_key = var.wg0_private_key
  enabled     = true
}
```

## 🧪 Running Tests

```sh
# Run offline unit tests (uses httptest mock server)
go test -v -race ./...

# Run live acceptance tests against an isolated test router/VM
TF_ACC=1 PFSENSE_URL="https://127.0.0.1:8443" PFSENSE_USERNAME="terraform-api" PFSENSE_PASSWORD="..." go test -v ./internal/provider/...
```
