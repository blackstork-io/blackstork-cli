# Plugin development

These instructions apply to `plugins/` and supplement the repository
`AGENTS.md`.

## Runner schemas

- A runner's `Doc` field is the introduction to its public reference page.
  Write it for template authors: state what the runner does, what it returns or
  renders, and any behavior needed to choose it correctly.
- Reconcile every description with the current `Config`, `Args`, accepted
  formatters, and implementation. Do not mention removed or inferred
  attributes.
- Write attribute documentation as direct guidance. Name accepted formats,
  units, defaults, limits, and interactions when they affect correct use.
- Mark credentials and tokens as `Secret` and use meaningful constraints for
  required values.
- Add an `ExampleVal` when a required attribute can have a safe, non-secret
  example.

## Implementation

- Keep vendor clients behind loader interfaces so runner behavior can be tested
  without live credentials.
- Preserve context cancellation and return actionable diagnostics without
  exposing secrets.
- Convert external values through `plugindata`; do not leak provider SDK types
  into the shared runtime.
- Keep runner names and block attributes backward compatible unless the change
  explicitly includes a migration.

## Validation

Run focused tests for the changed plugin, `go build ./plugins/...`, and plugin
schema validity checks when available. After changing a schema or description,
run `make generate-docs` and commit the corresponding `docs/plugins/` updates.
