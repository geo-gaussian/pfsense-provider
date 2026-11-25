# 🌐 Terraform Provider for pfSense Nexus

[![Unit coverage](https://img.shields.io/endpoint?url=https%3A%2F%2Fgeo-gaussian.github.io%2Fterraform-provider-pfsense-nexus%2Funit.json)](https://app.codecov.io/gh/geo-gaussian/terraform-provider-pfsense-nexus)
[![Live acceptance coverage unavailable](https://img.shields.io/badge/acceptance-unavailable-lightgrey)](#-coverage)
[![Nexus API coverage](https://img.shields.io/endpoint?url=https%3A%2F%2Fgeo-gaussian.github.io%2Fterraform-provider-pfsense-nexus%2Fapi.json)](#-coverage)

This experimental provider manages firewall aliases, NAT and filter rules,
schedules, traffic shapers, limiters, packet-flow exporters, virtual IPs, and
WireGuard settings, tunnels, and peers. It also reads selected firewall,
network, service, VPN, and system configuration or status from a local pfSense
firewall through the Nexus controller. WireGuard key generation is available
as an ephemeral resource. It uses Netgate's OpenAPI specification and
HashiCorp's Terraform Plugin Framework.

**Status:** Login and alias listing have been tested on one real appliance.
Firewall and WireGuard mutations, and read-only data sources other than alias
listing, have only been tested against mock servers and the included OpenAPI
specification. Do not use these resources on a production firewall. Verify API
behaviour, compatibility, and recovery in an isolated environment first.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the component and request lifecycle
overview.

## 🔥 Managed firewall resources

The provider manages firewall aliases, filter rules, schedules, NPT NAT,
one-to-one NAT, port-forward NAT, ALTQ root queues, limiters, PFlow exporters,
and virtual IPs. See the [resource documentation](docs/resources/) for
individual resource schemas and import instructions.

Outbound NAT and global PFlow options remain read-only: the API does not
provide the complete delete or reset lifecycle needed to manage them safely.
Firewall resource mutations initiate a filter reload but do not confirm its
completion.

The provider routes firewall and WireGuard operations through
`/api/device/pfsense/localhost/api`. This assumes the controller's local firewall
is registered as `localhost`; alias listing worked on one appliance, but other
installations have not been verified. Firewall resource mutations initiate a
filter reload but do not confirm its completion. WireGuard mutations use the
documented API operations without an additional apply call because the
included specification defines no WireGuard-specific apply endpoint. Requests
are serialised per configured client, not across separate provider
configurations or Terraform processes. If a write or reload fails, check the
firewall configuration and initiate the appropriate reload manually before
retrying: Terraform refresh may observe changed configuration without retrying
the failed reload. A failed create with an uncertain API response may require
importing the existing resource.

## 📋 Requirements

- Go 1.25 or newer to build
- Terraform 1.8 or newer
- Terraform 1.10 or newer to use the WireGuard ephemeral key-pair resource
- Node.js 22.18+ on the 22.x line, or 23.6+, and pnpm for
  repository lint checks
- jq for the Pages deployment guard and its offline tests
- A Nexus controller with a local firewall registered as described above

## 📥 Install from the Terraform Registry

Declare `geo-gaussian/pfsense-nexus` in `required_providers`, as shown below,
and run `terraform init` without a development override. Commit
`.terraform.lock.hcl` for reproducible provider selection; do not commit state,
saved plans, or files containing credentials.

## 🛠️ Build and configure for development

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

## 🔍 Read-only data sources

Read-only data sources expose selected firewall, network, service, VPN, and
system configuration or status as JSON; use `jsondecode()` to select fields.
Some outputs are sensitive, and the WireGuard status source omits key material.
For example, read aliases, configured interfaces, and firewall NAT rules:

The catalogue also includes DHCP lease and service status reads, system summary
and health information, gateway and interface status, and monitoring metadata.
Sensitive outputs are marked in their individual data source documentation.

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

data "pfsense_interfaces" "current" {}

data "pfsense_firewall_nat_port_forward" "current" {}

output "firewall_aliases" {
  value = data.pfsense_firewall_aliases.current.aliases
}

output "interfaces" {
  value = jsondecode(data.pfsense_interfaces.current.json)
}

output "port_forward_rules" {
  value = jsondecode(data.pfsense_firewall_nat_port_forward.current.json)
}
```

With the development override configured, run `terraform plan`. Data sources
perform API reads and do not write configuration or reload the firewall.
Authentication may create a session or audit record. Protect Terraform state
and saved plans, especially when using sensitive data sources.

The output contains the name, type, description (`descr`), and `truncated` flag
for user-defined aliases, not system aliases or alias contents. Fields omitted
by the API are null. If the API omits the alias list, the output is null and a
warning is shown; an empty list means the API returned an empty list. Saved
plans and state can contain alias metadata and other sensitive information.

## 📥 Importing aliases

Import an existing alias by name:

```sh
terraform import pfsense_firewall_alias.example example
```

The former `pfsense_nexus_firewall_alias` resource type is no longer supported.
If you have state from the earlier experimental version, back it up, remove
the old resource addresses from state (not from the firewall), and import the
existing aliases under the new resource type. Review the plan before applying.

## 📄 API specification and generation

[docs/nexus.openapi.yaml](docs/nexus.openapi.yaml) corresponds to
Netgate's [pfapi_openapi.yml](https://github.com/Netgate/pfsense-api/blob/b4acd7ca21dbca739a530403a1f62a7440f7726f/pfapi_openapi.yml)
at commit `b4acd7ca21dbca739a530403a1f62a7440f7726f`, with whitespace
changes. The upstream project is distributed under the Apache 2.0 licence; see
the included [licence](docs/NETGATE-LICENSE).

Run `make generate` to regenerate the Terraform alias schema and the Go HTTP
client using pinned generator versions. The generated `*_gen.go` files are not
hand-edited. The selected read-only Terraform data sources are custom
implementations; `make generate` does not generate their schemas or behaviour.
Terraform CRUD, data-source behaviour, session management, request
serialisation, and reload handling remain custom code.

## 💻 Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for implementation and testing playbooks
and [AGENTS.md](AGENTS.md) for mandatory project rules.

```sh
pnpm install --ignore-scripts
pnpm run setup-hooks
pnpm run lint
pnpm run test
```

The lint and format commands require [uv](https://docs.astral.sh/uv/);
they run a pinned YAPF version through `uvx`. Node runs the TypeScript
tooling directly; lint also checks its types without emitting build files.
The lint checks include a targeted inline-program safeguard, described in
[DEVELOPMENT.md](DEVELOPMENT.md). Spec Kit-managed scripts under
`.specify/scripts/` are exempt from the no-inline-program rule.

The tests check formatting and use mock HTTP servers; when the Terraform CLI is
available they also exercise create, update, import, refresh, and destroy
through the actual provider protocol. They do not replace live acceptance tests.
The Git hook and CI run the same lint and tests, with CI running lint and tests
in parallel. CI also produces coverage reports.

## 📊 Coverage

Run `pnpm run coverage` to run Go race tests and produce the following files
under `coverage-report/`:

- `unit.out`: Go statement coverage for project Go code, excluding files
  marked as generated and the coverage reporting tool itself. Inspect it with
  `go tool cover -func=coverage-report/unit.out` or
  `go tool cover -html=coverage-report/unit.out`.
- `api-report.md`: every HTTP method and path in the bundled OpenAPI
  specification, with implementation and mock-test evidence where supported.
  The percentage is implemented operations divided by all documented
  operations, not the percentage of API fields or behaviours implemented.
- `badges/`: Shields endpoint JSON for unit and API coverage, plus an
  unavailable acceptance badge.

Unit coverage includes in-process mock tests. The Terraform protocol tests use
mock HTTP servers and spawn a separate, uninstrumented provider process; they
are neither live acceptance tests nor a measurement of that process's
coverage. Manually observed API operations are identified separately in the
API report and do not imply live acceptance coverage. Live acceptance coverage
remains unavailable until real appliance tests and child-process coverage
collection are implemented. A future live job must build an instrumented
provider with `go build -cover`, collect its data using `GOCOVERDIR`, convert
it with `go tool covdata textfmt`, and upload that separate profile with an
`acceptance` Codecov flag. It must not reuse the unit profile.
Unit tests simulate some dependency failures that current implementations
cannot produce. Full statement coverage does not guarantee every branch is
tested or replace live acceptance testing.

CI uploads the filtered profile to Codecov with the `unit` flag and publishes
the reports as a workflow artifact. Connect the repository to Codecov and add
its upload token as the GitHub Actions secret `CODECOV_TOKEN`; protected
branches normally require it. Fork and Dependabot workflows may not receive
repository secrets, so uploads can be unavailable. Upload errors do not block
CI; inspect the upload step and Codecov to confirm delivery. Do not share the
token in issues, logs, or configuration files.

To publish the unit and API badges, enable GitHub Pages in repository settings
and select **GitHub Actions** as the source. After lint and tests pass on a
push to `main`, the workflow deploys badge data to Pages. Until the first
successful deployment, those badges may show an error. Badges reflect the
latest successful main-branch deployment, not pull requests. Shields and Pages
caches can delay updates. The badge colours are green at 95% or above,
yellow-green at 80%, yellow at 70%, orange at 50%, and red below 50%;
unavailable is grey.
Percentages are truncated to two decimal places for display; colour thresholds
use the unrounded percentage.

When adding provider support, update the explicit operation evidence in
`tools/coverage/main.go` with the matching method, path, provider call, and
mock-test file. The report checks mappings against the specification and
checks that the provider call exists; maintainers must verify that the
referenced tests actually exercise each operation. Generated client methods
alone do not count as support.

Future acceptance tests can run as a separate workflow job against an isolated,
disposable pfSense instance with test-only credentials. If they become a release
requirement, add that job to the release job's dependencies and ensure it runs
on version tags; a skipped dependency would otherwise prevent the release. Do
not expose test credentials or an appliance to untrusted pull request code.

The repository prefers tabs displayed at two columns for project-owned code
and JSON. YAML, Terraform, and Markdown use two-space indentation; Go follows
`gofmt` and Makefile recipes use tabs. Lint uses YAPF for Python and Prettier
for Node scripts, TypeScript, JSON, and shell, including Spec Kit's managed
files. Spaces
are allowed for alignment and content where needed. Markdown and YAML are not
automatically formatted; the whitespace check catches some issues but does not
enforce all indentation. Project-owned shell scripts use POSIX shell and
`#!/bin/sh`; Spec Kit-managed scripts are temporarily exempt. POSIX shell
compatibility is not currently checked by lint.

Run `pnpm run format` to reformat those files after generating or updating
Spec Kit. Review the changes before committing. Formatting and skill changes to
managed files count as local edits: Spec Kit upgrades may preserve them or
require manual reconciliation. Do not update its recorded hashes to hide
changes.

## 📦 Preparing a release

The build and deploy workflow runs lint and tests on pushes and pull requests.
For a pushed version tag, it creates a signed **draft** GitHub release only
after validation succeeds. It does not publish the draft automatically. Before
tagging:

1. Register the GPG signing public key in the Terraform Registry. Configure
   the GitHub Actions secrets `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE`; never commit
   or share the private key or passphrase.
2. Verify the release documentation and experimental warnings, and ensure the
   intended commit has passed checks. The workflow builds Linux, macOS, Windows,
   and FreeBSD packages for AMD64 and ARM64.
3. Push a new version tag. Review the draft release and its
   signed checksums before publishing it. Do not replace assets of an already
   published release.
4. Register the repository in the Terraform Registry, or let its webhook ingest
   later published releases. After ingestion, test `terraform init` in a fresh
   directory without development overrides.

The workflow embeds the tag version in the provider binary. Local builds report
`dev`. Signing and publishing require repository and Registry permissions; the
workflow alone does not configure those permissions or register the provider.
