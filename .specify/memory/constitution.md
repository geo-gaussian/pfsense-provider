<!--
Sync Impact Report
Version change: 1.2.0 → 2.0.0
Modified principles: none.
Modified sections: Quality Workflow and Gates.
Added sections: none.
Removed sections: none.
Follow-up TODO: none.
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

Contributors MUST run `pnpm run coverage` and compare the reported unit
statement-coverage ratio with the base revision using the same tooling and
coverage rules. A change MUST NOT lower coverage unless its pull request gives
a clear justification for the decrease and receives explicit reviewer
approval. When statement totals differ, compare ratios rather than covered
counts alone. The command does not enforce this comparison, so the pull request
MUST report both values and include any required justification.

When a defensive error path cannot be reached through a valid Terraform schema,
isolate the conversion boundary so the error behaviour can be tested without
weakening production error handling. Coverage is a regression guard, not a
substitute for meaningful assertions.

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
- Hand-written production files MUST NOT exceed 500 physical lines and their
  functions MUST NOT exceed 80 physical lines. Exceeding 350 lines per file or
  40 lines per function MUST produce a warning.
- Test files MUST NOT exceed 600 physical lines and their functions, including
  helpers, MUST NOT exceed 150 physical lines. Exceeding 500 lines per file or
  100 lines per function MUST produce a warning.
- Count comments and blank lines. Count a function from its declaration through
  its closing brace, including nested closures. Generated code, fixtures,
  lockfiles, and API specifications are excluded from these size limits.
- Any exception for hand-written code MUST identify the specific file or
  function, explain its rationale, and receive project review.

## 📋 Quality Workflow and Gates

- Spec Kit implementation plans MUST include a Constitution Check that covers
  the principles above. Behaviour-changing work MUST include test tasks even
  when a generic Spec Kit template describes tests as optional.
- Before completion, run `pnpm run coverage`, compare its report with the base
  revision, and run `pnpm run lint`. Any coverage decrease MUST have a
  documented justification and explicit reviewer approval.
- Run `go test -race ./...` when the coverage command has not already run it.
- Review tests for meaningful assertions, representative error cases, and
  absence of dependence on a live appliance.
- File and function size errors MUST prevent commits and merges. Individual
  warnings MUST be reported, and three or more warnings across lint commands
  MUST prevent commits and merges. Warning-producing lint tools MUST use a
  recognised warning format or an adapter so their warnings are counted.
- Any intentional exception MUST be documented in the change and approved
  through project review; exceptions MUST NOT silently lower coverage.

## ⚖️ Governance

This constitution governs project decisions and takes precedence over generic
Spec Kit guidance when they conflict. Amendments MUST be made through review,
include a rationale, and update the version and amendment date.

Use semantic versioning for this constitution: MAJOR for incompatible principle
changes, MINOR for new principles or materially expanded requirements, and PATCH
for clarifications that do not change obligations. Reviewers MUST check proposed
changes against these principles and the current `AGENTS.md` and `DEVELOPMENT.md`
guidance.

**Version**: 2.0.0 | **Ratified**: 2026-11-26 | **Last Amended**: 2026-11-26
