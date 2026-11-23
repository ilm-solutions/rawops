# RAWOPS — Linux Infrastructure Operations CLI

RAWOPS is an early-stage Go CLI foundation for Linux host operations: configuration validation, diagnostics, performance policy planning, and host security management.

## Current implementation status

Implemented in this foundation:
- Cobra CLI with non-zero exit codes for failed commands.
- `rawops status` reports detected host capabilities rather than simulated readiness.
- `rawops doctor` checks prerequisites without modifying the host.
- `rawops init-config` creates a private starter config and refuses to overwrite.
- `rawops apply` parses and validates the supported TOML subset and prints a plan. It does **not** claim to apply settings.
- `rawops shield` generates a dedicated nftables table by default. `--apply` validates the policy and snapshots RAWOPS-owned files before applying; it never uses `flush ruleset`.
- Atomic file-write helper and timestamped RAWOPS-owned snapshots.

Not yet production-ready:
- Full TOML compatibility, transactional multi-subsystem apply, complete rollback, agent/control-plane, remote fleet management, RBAC, signed updates, packaging and CI release pipeline.
- The firewall policy is a starter host policy, not a complete perimeter firewall or DDoS mitigation product. Review it for your host's access requirements before use.
- The current snapshot helper records RAWOPS-managed config files; it is not a full machine backup.

## Build and test

Requires Go 1.23 or newer (as specified in `go.mod`).

```sh
go test ./...
go build -o rawops .
./rawops --help
./rawops doctor
./rawops status
./rawops init-config
./rawops apply --config rawops.toml
./rawops shield --ports 22,25565
```

Only run host-changing commands on systems you administer, after reviewing the generated policy and maintaining console or out-of-band access.

## Architecture direction

```text
CLI (Cobra)
  ├── Diagnostics / Doctor
  ├── Declarative Config + Validation
  ├── Policy Planning
  ├── Safe Execution Adapter (timeouts, errors, explicit privilege)
  ├── Host Snapshot Store (RAWOPS-owned state only)
  └── Domain modules (performance, firewall, telemetry, provisioning)
```

Future enterprise layers should be introduced behind stable interfaces:
1. Domain model and validation packages.
2. Idempotent plan/apply executor with preflight, snapshot, health check and rollback.
3. Structured audit log and machine-readable output.
4. Host agent and authenticated control API (only if remote management is later selected).
5. Release engineering: CI, tests, SBOM, signed artifacts and upgrade/rollback policy.
