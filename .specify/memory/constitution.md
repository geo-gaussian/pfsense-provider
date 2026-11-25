<!--
Sync Impact Report
Version change: unratified template → 1.0.0
Modified principles: none; the prior file contained only template placeholders.
Added sections: five project principles, engineering constraints, quality workflow and governance.
Removed sections: none.
Follow-up TODO: confirm the original ratification date.
-->

# 🏛️ pfSense Nexus Provider Constitution

## 🧭 Core Principles

### 🧱 I. Contract-Faithful Provider

Provider behaviour MUST match the Nexus API contract in `docs/nexus.openapi.yaml`.
Review request paths, methods, payloads, response handling, and endpoint side effects
before implementation. This keeps Terraform state and appliance behaviour aligned.

### 🧱 II. Safe and Complete Terraform Lifecycles

Every managed resource MUST implement its applicable lifecycle methods and import
behaviour. Reads MUST handle out-of-band deletion without an error diagnostic, and
write operations MUST verify success. When an API masks or omits a secret, reads
MUST retain its prior state value. Fields containing secrets MUST be marked
sensitive.

### 🧱 III. Regression Tests and Coverage

Every change to provider behaviour MUST include focused tests for the changed
behaviour, including relevant success, validation, API failure, malformed-response,
and state-preservation paths. Tests MUST assert outcomes and diagnostics rather
than merely execute lines.

The unit statement-coverage target is 100% of statements included by the
repository's coverage report. Contributors MUST run `pnpm run coverage` and
preserve that target. When a defensive error path cannot be reached through a
valid Terraform schema, isolate the conversion boundary so the error behaviour
can be tested without weakening production error handling. Coverage is a
regression guard, not a substitute for meaningful assertions.

### 🧱 IV. Explicit API and Code-Generation Boundaries

All pfSense HTTP requests MUST use the configured `NexusClient.DoRequest` path.
Generated API clients and custom Terraform lifecycle logic MUST remain distinct;
regeneration MUST preserve custom resource and data-source code and registration.
Operations that stage changes MUST trigger their required apply or reload operation
after create, update, and delete.

### 🧱 V. Honest Documentation and Compatibility

Project documentation MUST use British English and describe tested behaviour
accurately. Mock-server evidence MUST NOT be presented as live appliance
compatibility. Documented examples MUST contain no credentials or other sensitive
values.

## 📋 Additional Engineering Constraints

- Project-owned shell scripts MUST use POSIX shell syntax and `#!/bin/sh`.
- Executable programs and reusable test fixtures MUST live in standalone files;
  do not embed programs in shell commands, workflow blocks, or other languages.
- Follow `AGENTS.md` for detailed implementation constraints, secret-handling
  rules, and repository-specific conventions.
- Use the repository's pinned generators and tooling; do not make unrelated
  generated-file or dependency changes.

## 📋 Quality Workflow and Gates

- Spec Kit implementation plans MUST include a Constitution Check that covers
  the principles above. Behaviour-changing work MUST include test tasks even
  when a generic Spec Kit template describes tests as optional.
- Before completion, run `pnpm run coverage` and confirm the report remains at
  100% unit statement coverage, then run `pnpm run lint`.
- Run `go test -race ./...` when the coverage command has not already run it.
- Review tests for meaningful assertions, representative error cases, and
  absence of dependence on a live appliance.
- Any intentional exception MUST be documented in the change and approved
  through project review; exceptions MUST NOT silently lower the coverage target.

## ⚖️ Governance

This constitution governs project decisions and takes precedence over generic
Spec Kit guidance when they conflict. Amendments MUST be made through review,
include a rationale, and update the version and amendment date.

Use semantic versioning for this constitution: MAJOR for incompatible principle
changes, MINOR for new principles or materially expanded requirements, and PATCH
for clarifications that do not change obligations. Reviewers MUST check proposed
changes against these principles and the current `AGENTS.md` and `DEVELOPMENT.md`
guidance.

**Version**: 1.0.0 | **Ratified**: TODO | **Last Amended**: 2026-11-25
