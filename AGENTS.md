# 🤖 AGENTS.md — AI Coding Agent Guidelines

## 🎯 Project Context

This repository implements an experimental Terraform provider in Go, published
as `geo-gaussian/pfsense-nexus`. It wraps the Nexus API described in
`docs/nexus.openapi.yaml`. It has experimental firewall and WireGuard resources,
resource discovery, selected read-only data sources, and a WireGuard ephemeral
key-pair resource. Login and alias listing have been tested on one appliance;
resource writes, discovery, key generation, and additional data sources have
only been tested with mock servers. Compatibility with other appliance versions
remains unverified.

## 📐 Architectural Rules & Constraints

1. **Never Bypass `NexusClient.DoRequest`:** All HTTP calls to pfSense MUST route through `internal/client.NexusClient.DoRequest()`. Its mutex serialises HTTP requests per configured client, not across separate clients or Terraform processes.
2. **Preserve Generated Files vs. Custom Logic:**
   - HashiCorp's OpenAPI resource generator produces schemas for the firewall
     resources mapped in `generator_config.yaml`. Their lifecycle code is
     custom, as are the WireGuard resources, ephemeral key-pair resource, list
     resources, and read-only data sources. `oapi-codegen` generates the API
     client methods. When regenerating, verify generated files and preserve
     custom lifecycle code and registration.
3. **Explicit Apply / Reload Lifecycle:**
   - If an endpoint in `docs/nexus.openapi.yaml` stages changes without immediately activating them, trigger its corresponding apply or reload operation after `Create`, `Update`, and `Delete`. The alias resource initiates a filter reload but does not confirm completion.
4. **Strict Secret Handling:**
   - Any field holding keys, passwords, PSKs, or tokens (`private_key`, `privatekey`, `private_key_clamped`, `preshared_key`, `presharedkey`, `password`, `tls_key`) MUST have `Sensitive: true` in its `schema.StringAttribute`.
   - If a `GET` endpoint masks or omits a secret value, `Read()` MUST retain the existing `state` value rather than overwriting it with an empty string.
5. **Graceful 404 Handling in `Read()`:**
   - When the Nexus API returns `HTTP 404 Not Found` during `Read()`, call `resp.State.RemoveResource(ctx)` and return without adding an error diagnostic so Terraform detects out-of-band deletion cleanly.
6. **POSIX Shell Scripts:**
   - Project-owned shell scripts MUST use POSIX shell syntax and `#!/bin/sh`. Spec Kit-managed scripts are temporarily exempt.
7. **No Inline Programs:**
   - Store executable code, configuration used as a program, and test fixtures in standalone files in their own language. Pass dynamic values as arguments, environment variables, or supported configuration variables. Do not embed programs in shell commands, workflow blocks, other languages' strings, or agent terminal commands.
   - This includes interpreter `-c`/`-e` code and inline AWK, sed, and jq programs. Ordinary commands and arguments are allowed.
   - Spec Kit-managed scripts under `.specify/scripts/` are exempt. No other automatic exception applies, including tests and generated fixtures: obtain the user's explicit approval before adding any other exception.
   - Short, syntax-highlighted documentation examples may explain a concept; reusable automation belongs in standalone files.

## 📝 Documentation Rules

- Markdown headings at levels 1, 2, and 3 MUST include a relevant emoji.
- Use `.yaml` for project-owned YAML files. Keep `.yml` only where an external
  tool requires that exact filename.
- Use two-space indentation in Markdown and preserve rendered list and code
  block structure.
- Write project documentation in British English.
- Follow the repository's cspell configuration and update documentation when
  behaviour or development workflows change.
- Keep examples free of credentials and other sensitive information.

## 📏 Maintainability Limits

- Count physical lines, including blank lines and comments. Function spans
  include the declaration, end of the definition, and any nested closures.
- For hand-written production code, warn above 350 lines per file or 40 lines
  per function; block above 500 lines per file or 80 lines per function.
- For test code, including test helpers, warn above 500 lines per file or 100
  lines per function; block above 600 lines per file or 150 lines per function.
- Exclude generated code, fixtures, lockfiles, and API specifications.
  Exceptions for hand-written code MUST identify the exact file or function,
  explain why an exception is needed, and be approved in project review.
- Size warnings MUST NOT block commits or merges; errors MUST block both.
  See `DEVELOPMENT.md` for checker scope and its current jq function limitation.

## 🔄 Standard Development Workflow

Use pnpm for Node-based repository tooling. The repository `.npmrc`
disables package installation scripts; do not override this setting. Run
required project scripts explicitly with `pnpm run`.

```mermaid
sequenceDiagram
    participant Dev as Agent / Developer
    participant Gen as OpenAPI code generators
    participant Impl as Provider implementation
    participant Client as internal/client (Mutex + Auth)
    participant Test as go test + coverage

    Dev->>Gen: 1. Update generator mappings or API operation IDs; regenerate
    Gen->>Impl: 2. Generate resource schemas and selected API client methods
    Dev->>Impl: 3. Implement resource lifecycle or custom read-only data source
    Impl->>Client: Route every HTTP request through NexusClient.DoRequest
    Dev->>Test: 4. Add mock HTTP and provider-protocol tests as applicable
    Test-->>Dev: 5. Compare coverage with the base revision; justify any decrease, then lint
```

## 🧪 Test Quality and Coverage

- Every behaviour change MUST include focused regression tests for the changed
  behaviour. Cover success and relevant validation, API failure, malformed or
  missing response, state-preservation, and out-of-band deletion paths.
- Tests MUST assert expected state, outputs, and diagnostics. Do not add tests
  solely to execute statements or weaken runtime checks to improve coverage.
- Run `pnpm run coverage` and compare its unit statement-coverage ratio with the
  base revision using the same tooling. Compare ratios when the statement totals
  differ; a larger covered-statement count alone does not prove coverage did
  not regress.
- If coverage decreases, the pull request MUST explain the cause, impact, and
  risk mitigation, and receive explicit reviewer approval. The command does not
  enforce this comparison, so report the before-and-after values in the pull
  request. New or changed behaviour still requires meaningful tests.
- If a defensive error path cannot be reached with a valid Terraform schema,
  isolate its conversion boundary and test the error behaviour without
  weakening production error handling.
- `pnpm run coverage` already runs `go test -race ./...`; run that Go command
  separately only when race testing is needed without a coverage report.

## ✅ Definition of Done for Any New Resource

Before marking a resource complete, verify:

- [ ] Endpoint paths and JSON field tags match `docs/nexus.openapi.yaml`.
- [ ] `Create`, `Read`, `Update`, `Delete`, and `ImportState` are all implemented.
- [ ] Sensitive fields are marked `Sensitive: true`.
- [ ] Mock HTTP tests cover successful lifecycle behaviour and relevant
  validation, API-error, malformed-response, and secret-preservation paths.
- [ ] `pnpm run coverage` has been compared with the base revision; any decrease
  has a reviewed justification. `pnpm run lint` passes.
- [ ] No hardcoded credentials, IPs, or private keys appear in code or examples.

## 📖 Definition of Done for Any New Data Source

- [ ] The OpenAPI operation is a safe read and its request, response, and endpoint behaviour have been reviewed.
- [ ] The selected operation ID is in `oapi-codegen.yaml`, and the custom data source is registered in `internal/provider/provider.go`.
- [ ] Sensitive or identifying JSON output is marked sensitive; secret material is omitted from status projections where appropriate.
- [ ] A data-source page exists under `docs/data-sources/`, and mock HTTP tests
  cover the endpoint and relevant failures.
- [ ] `pnpm run coverage` has been compared with the base revision; any decrease
  has a reviewed justification. `pnpm run lint` passes, and live appliance
  compatibility is clearly described as unverified unless tested.
