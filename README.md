# RAWOPS — Linux Infrastructure Operations CLI

RAWOPS is a CLI-only Go foundation for Linux host operations. Its intended direction is a hybrid bare-metal/VPS control plane that integrates existing Linux infrastructure tools rather than reimplementing them.

## Supported target and current scope

- Primary target: Linux x86-64 (AMD EPYC, Ryzen, Intel Xeon) on supported Debian-, Ubuntu-, and RHEL-family systems.
- ARM64 builds can compile and report that x86-64 CPUID assembly is bypassed; ARM64 is not currently a production target.
- VPS/cloud and bare-metal are architectural targets. Provider APIs and remote fleet management are not implemented in this release.
- Interface: CLI only. No GUI is planned.

## Current implementation status

Implemented foundations:
- Cobra CLI with non-zero exit codes for command errors.
- `rawops status` reports basic host facts and tool paths.
- `rawops doctor` checks Linux/x86-64 prerequisites without changing host state.
- `rawops init-config` creates a private starter config without overwriting an existing path.
- `rawops apply` validates the supported configuration subset and prints a plan. It does **not** apply host settings.
- `rawops shield` generates a dedicated RAWOPS nftables table by default. Live apply is root-gated, syntax-preflighted, snapshotted, and applied as an nftables batch. It does not flush unrelated tables.
- Shield policy currently uses `policy accept`; it is additive and is **not** a default-deny inbound firewall or DDoS mitigation system.
- Snapshot state is limited to RAWOPS-managed sysctl configuration and the RAWOPS nftables table; it does not capture arbitrary runtime kernel state or the whole host. Rollback validates snapshot state and checksums and refuses to proceed if nftables is unavailable. A failure between sysctl and firewall restoration can still leave a partially restored host, and is reported explicitly.
- `rawops tune` generates reviewable profile files. It does not change kernel settings; `--apply` is intentionally rejected.
- `rawops monitor` displays aggregate CPU utilization, memory, aggregate packet-counter telemetry, and the CPU governor when available. It is not a full monitoring or alerting system.
- `rawops arsenal list` and `rawops arsenal bundle` show a 42-module catalogue and five presets. `rawops arsenal check` checks for module executables (read-only); it does **not** install or configure packages.
- AMD64 CPUID assembly returns the CPU vendor identifier; it does not identify the CPU model or all hardware capabilities.

## Explicitly not production-complete

- Transactional multi-subsystem apply and fully atomic cross-subsystem rollback.
- Complete host firewall policy, adaptive traffic mitigation, or DDoS protection.
- Package installation, service configuration, dependency orchestration, and verification for Arsenal modules.
- Full TOML compatibility, hardware-aware performance optimization, advanced monitoring/alerting, remote fleet management, RBAC, signed updates, and provider integrations.
- A complete machine backup. RAWOPS snapshots only RAWOPS-owned state.

Live host-changing commands should only be used by an authorized administrator on a supported Linux server after reviewing the generated policy and ensuring console or out-of-band recovery access. Android/Termux is suitable for source editing and non-privileged tests, not live firewall or rollback validation.

## Build and test

Requires Go 1.23 or newer.

```sh
gofmt -w .
go test ./...
go vet ./...
go build -trimpath -o rawops .
./rawops --help
./rawops doctor
./rawops status
./rawops arsenal check
./rawops monitor --ticks 5
```

## Architecture direction

```text
CLI (Cobra)
  ├── Core: config, capability checks, validation, planning, authorization
  ├── Domain modules: host, network, shield, security, monitor, performance
  ├── Recovery: RAWOPS-owned snapshots and verified restore
  ├── Arsenal: external-tool catalogue and future package/service adapters
  └── Platform adapters: supported Linux distributions and hardware
```

A future operation executor should follow:

```text
Discover -> Validate -> Authorize -> Plan -> Preflight -> Snapshot
         -> Execute -> Verify -> Record
```

Host-changing actions should remain opt-in, scoped to RAWOPS-owned state, auditable, and fail closed when prerequisites or recovery checks are unavailable.
