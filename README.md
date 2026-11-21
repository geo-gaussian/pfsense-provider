# Terraform Provider for pfSense Nexus

This experimental provider manages a firewall alias on a local pfSense
firewall through the Nexus controller. It uses Netgate's OpenAPI specification
and HashiCorp's Terraform Plugin Framework.

**Status:** Login and alias listing have been tested on one real appliance.
Alias writes and reloads have only been tested against mock servers. Do not use
the resource on a production firewall. Verify API behaviour, compatibility,
and recovery in an isolated environment first.

The provider routes firewall operations through
`/api/device/pfsense/localhost/api`. This assumes the controller's local firewall
is registered as `localhost`; alias listing worked on one appliance, but other
installations have not been verified. Firewall writes initiate a filter reload,
but do not establish that the resulting rules are active. A mutex serializes requests within one
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
    pfsense = {
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

provider "pfsense" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
}

resource "pfsense_firewall_alias" "example" {
  name     = "example"
  type     = "host"
  targets  = [{ name = "host.example.test" }]
}
```

The local provider name `pfsense` maps to the `geo-gaussian/pfsense-nexus`
source. Pass credentials through a secure mechanism, such as Terraform's environment
variables, rather than committing them. TLS certificates are verified by
default; `insecure = true` disables verification and is for isolated testing
only. Terraform state can contain sensitive configuration; protect it.

## Read-only alias lookup

To test authentication and read alias metadata without managing an alias, use a
separate Terraform directory with only this configuration:

```hcl
terraform {
  required_providers {
    pfsense = {
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

provider "pfsense" {
  url      = var.nexus_url
  username = var.nexus_username
  password = var.nexus_password
}

data "pfsense_firewall_aliases" "current" {}

output "firewall_aliases" {
  value = data.pfsense_firewall_aliases.current.aliases
}
```

With the development override configured, run `terraform plan`. This data
source authenticates and requests the local firewall's alias list; it does not
create, update, delete, or reload aliases. Login may create a session or audit
record. A successful read verifies the login and list request, but the routing
assumption above may not hold for other installations.

The output contains the name, type, description (`descr`), and `truncated` flag
for user-defined aliases, not system aliases or alias contents. Fields omitted
by the API are null. If the API omits the alias list, the output is null and a
warning is shown; an empty list means the API returned an empty list. Saved
plans and state can contain alias metadata and other sensitive information.

## Importing aliases

Import an existing alias by name:

```sh
terraform import pfsense_firewall_alias.example example
```

The former `pfsense_nexus_firewall_alias` resource type is no longer supported.
If you have state from the earlier experimental version, back it up, remove
the old resource addresses from state (not from the firewall), and import the
existing aliases under the new resource type. Review the plan before applying.

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

## 📦 Preparing a release

The Registry release workflow runs on a pushed version tag, runs lint and tests,
and creates a signed **draft** GitHub release. It does not publish the draft
automatically. Before tagging:

1. Register the GPG signing public key in the Terraform Registry. Configure
   the GitHub Actions secrets `GPG_PRIVATE_KEY` and `PASSPHRASE`; never commit
   or share the private key or passphrase.
2. Verify the release documentation and experimental warnings, and ensure the
   intended commit has passed checks. The workflow builds Linux, macOS, Windows,
   and FreeBSD packages for AMD64 and ARM64.
3. Push a version tag such as `v0.1.0`. Review the draft release and its
   signed checksums before publishing it. Do not replace assets of an already
   published release.
4. Register the repository in the Terraform Registry, or let its webhook ingest
   later published releases. After ingestion, test `terraform init` in a fresh
   directory without development overrides.

The workflow embeds the tag version in the provider binary. Local builds report
`dev`. Signing and publishing require repository and Registry permissions; the
workflow alone does not configure those permissions or register the provider.
