# RAWOPS — Bare-Metal Provisioning, Hardware Tuning & Security Engine

**RawOps** is a zero-bloat, CLI-only bare-metal control plane engineered in **Golang** and **x86-64 Assembly**. Designed for game hosting startups, VPS providers, and infrastructure operators who demand direct silicon-level control without GUI overhead.

## Core Architecture
- **Zero-Dependency Static Binary:** ~2.5 MB standalone `x86-64` Linux executable.
- **`rawops arsenal`:** Unified registry and automated deployment for core networking, virtualization, hardware, and security tools.
- **`rawops tune`:** Direct `CPUID` assembly hardware probing and custom kernel `sysctl` / TCP BBR performance profiles.
- **`rawops shield`:** Zero-trust `nftables` firewall and L3/L4 SYN-flood DDoS mitigation ruleset generator.

## Quick Start (x86-64 Linux)
```bash
sudo ./install.sh
rawops status
rawops arsenal list
sudo rawops tune --profile=game --apply
sudo rawops shield --mode=strict --ports=22,25565 --apply
```
