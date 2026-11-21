# Terraform Provider for pfSense Nexus

This experimental provider manages a firewall alias on a local pfSense
firewall through the Nexus controller. It uses Netgate's OpenAPI specification
and HashiCorp's Terraform Plugin Framework.

**Status:** The implementation has only been tested against mock servers, not a
real pfSense instance. Do not use it on a production firewall. Verify API
behaviour, compatibility, and recovery in an isolated environment first.

The provider routes firewall operations through
`/api/device/pfsense/localhost/api`. This assumes the controller's local firewall
is registered as `localhost`; that assumption has not yet been verified on an
actual appliance. Firewall writes initiate a filter reload, but do not establish
that the resulting rules are active. A mutex serializes requests within one
provider process; it does not coordinate separate Terraform processes.
If a write or reload fails, check the firewall configuration and initiate a
reload manually before retrying: Terraform refresh may observe the changed
configuration without retrying the failed reload. A failed create with an
uncertain API response may also require importing the existing alias.

## Requirements

- Go 1.25 or newer to build
- Terraform 1.8 or newer
- Node.js and pnpm for repository lint checks
- A Nexus controller with a local firewall registered as described above

## Build and configure Terraform

```sh
mkdir -p bin
go build -o bin/terraform-provider-pfsense-nexus .
```

In a development-only Terraform CLI configuration, set the override to the
**directory** containing that binary:

```hcl
provider_installation {
  dev_overrides {
    "geo-gaussian/pfsense-nexus" = "/absolute/path/to/bin"
  }
  direct {}
}
```

Set `TF_CLI_CONFIG_FILE` to that configuration file. Terraform development
overrides do not require `terraform init`.

```hcl
terraform {
  required_providers {
    pfsense-nexus = {
      source = "geo-gaussian/pfsense-nexus"
    }
  }
}

variable "nexus_url" { type = string }
variable "nexus_username" { type = string }
variable "nexus_password" {
  type      = string
  sensitive = true
}

provider "pfsense-nexus" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
}

resource "pfsense_nexus_firewall_alias" "example" {
  provider = pfsense-nexus
  name     = "example"
  type     = "host"
  targets  = [{ name = "host.example.test" }]
}
```

The explicit resource `provider` selection is needed because the provider
source name contains a dash while the resource type uses `pfsense_nexus`.
Pass credentials through a secure mechanism, such as Terraform's environment
variables, rather than committing them. TLS certificates are verified by
default; `insecure = true` disables verification and is for isolated testing
only. Terraform state can contain sensitive configuration; protect it.

Import an existing alias by name:

```sh
terraform import pfsense_nexus_firewall_alias.example example
```

## API specification and generation

[docs/nexus.openapi.yaml](docs/nexus.openapi.yaml) corresponds to
Netgate's [pfapi_openapi.yml](https://github.com/Netgate/pfsense-api/blob/b4acd7ca21dbca739a530403a1f62a7440f7726f/pfapi_openapi.yml)
at commit `b4acd7ca21dbca739a530403a1f62a7440f7726f`, with whitespace
changes. The upstream project is Apache-2.0 licensed; see the included
[license](docs/NETGATE-LICENSE).

Run `make generate` to regenerate the Terraform alias schema and the Go HTTP
client using pinned generator versions. The generated `*_gen.go` files are not
hand-edited. Terraform CRUD, session management, request serialization, and
reload handling remain custom code.

## Development

```sh
pnpm install --ignore-scripts
pnpm run setup-hooks
pnpm run lint
pnpm run test
```

The tests use mock HTTP servers; when the Terraform CLI is available they also
exercise create, update, import, refresh, and destroy through the actual
provider protocol. They do not replace live acceptance tests. The Git hook and
CI run the same lint and race-test commands.

The repository uses two-space indentation by default. Go source follows `gofmt`
and Makefile recipes use tabs.
