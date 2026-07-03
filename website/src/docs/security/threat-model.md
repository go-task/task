---
title: Threat Model
outline: deep
---

# Threat Model

This document outlines the security threats, assets, and mitigations for the
Task project. It serves as a high-level, public guide and is published as part
of our commitment to transparency.

## Asset Inventory

### Critical Assets

- **Source Code:** The Task CLI, build scripts, and configuration files
  (e.g., `Taskfile.yml`, `.goreleaser.yml`).
- **Build Artifacts:** Compiled binaries, packages, and containers distributed
  to users.
- **Secrets:** API tokens, signing keys, and repository credentials used in
  CI/CD and release pipelines.
- **Release Metadata:** Version numbers, changelogs, and checksums.
- **CI/CD Pipelines & Runners:** GitHub Actions workflows that build, test, and
  release the project.
- **Third-party Dependencies:** Go modules and tools used to build and
  distribute Task.
- **Website & Documentation:** The taskfile.dev site and installation scripts.

### Asset Locations

- Local developer machines
- GitHub Actions runners
- GitHub Releases
- Public package registries (npm, Homebrew, Winget, Cloudsmith)
- Source control platforms (GitHub)
- Netlify (website hosting)

## Threat Model

### Actors

- **Maintainers & Contributors:** Trusted users with varying levels of
  repository access.
- **External Attackers:** Untrusted users seeking to compromise builds,
  releases, or user systems.
- **Supply Chain Threats:** Malicious dependencies or compromised third-party
  services.
- **CI/CD Systems:** Automated agents that may be exploited if misconfigured.

### Entry Points

- Source code contributions (pull requests, issues)
- Configuration files and build scripts
- CI/CD integration and environment variables
- Third-party dependencies
- Release pipelines and artifact repositories
- Remote Taskfile fetching (HTTP, Git)
- Installation scripts

### Trust Boundaries

- Between the project repository and the CI/CD environment
- Between Task and remote Taskfiles fetched over the network
- Between artifact generation and distribution channels
- Between the Task binary and user-defined shell commands

### Threats

#### Supply Chain Attacks

- Compromised Go dependencies or build tools
- Unauthorized changes to source code or configuration
- Exploitation of third-party CI/CD or package registry services
- Compromised installation scripts or distribution channels

#### Secrets Leakage

- Exposure of tokens, credentials, or signing keys in logs, error messages,
  or artifacts
- Hardcoded secrets in code or configuration
- Improper secret management in CI/CD environments

#### Code Execution / Injection

- Malicious code execution via compromised pull requests or dependencies
- Remote code execution vulnerabilities in Task or its dependencies
- **Note:** Task intentionally executes user-defined shell commands as part of
  its core functionality. Users are responsible for the commands they define in
  their Taskfiles.

#### Unauthorized Access

- Unauthorized users triggering releases or accessing sensitive artifacts
- Insecure permissions on runners, repositories, or artifact stores
- Compromised maintainer accounts

#### Data Integrity & Tampering

- Tampering with build artifacts, changelogs, or metadata
- Compromise of signing keys, leading to malicious releases
- Man-in-the-middle attacks against remote Taskfile fetching

#### Denial of Service

- Abuse of CI/CD resources, bandwidth, or artifact storage
- Overloading automated processes or API endpoints
- Malicious Taskfiles designed to exhaust system resources

## Mitigations

### Supply Chain Security

- Pin dependencies and use trusted sources
- Mandatory code review and CI checks on all incoming pull requests
- Signed commits and release tags
- Enable immutable releases where supported
- Run `govulncheck` on every commit and tag
- Pin GitHub Actions to specific commit SHAs

### Secrets Management

- Secure storage using GitHub Secrets
- Never log or expose secrets in build or release outputs
- Regularly rotate secrets and monitor for suspicious activity
- Use least-privilege tokens scoped to specific repositories

### Secure Code Execution

- Validate and sanitize configuration files and user inputs
- Audit dependencies for vulnerabilities
- HTTP is rejected for remote Taskfiles by default (requires `--insecure` flag)
- TLS certificate verification for remote Git repositories

### Access Control

- Enforce least privilege for CI/CD runners, repositories, and artifact stores
- Require multi-factor authentication for maintainers
- Restrict release triggers to tagged releases only
- Lower permissions of less active maintainers

### Artifact Integrity

- Generate checksums for all release artifacts
- Distribute artifacts via trusted, access-controlled repositories
- Verify signatures and checksums in installation scripts where possible

### Availability Protection

- Implement rate limiting and resource quotas on CI/CD jobs
- Monitor for abnormal activity and automate alerts
- Set timeouts on network operations (e.g., remote Taskfile fetching)

## Residual Risks

- Zero-day vulnerabilities in dependencies, CI/CD systems, or Task itself
- Social engineering attacks targeting maintainers
- Unnoticed supply chain compromises
- Human error in configuration or secret management
- Users fetching malicious remote Taskfiles from untrusted sources

## Security Best Practices

- Regularly update dependencies and build tools
- Monitor security advisories and patch vulnerabilities promptly
- Educate contributors on secure coding and secrets hygiene
- Document security policies and incident response procedures

## References

- [Task Documentation](https://taskfile.dev/)
- [Incident Response Plan](./incident-response-plan)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Supply Chain Security](https://slsa.dev/)
- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
