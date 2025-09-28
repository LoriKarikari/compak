# compak - Project Proposal (Updated)

**TO:** L7 Engineer  
**FROM:** Project Lead  
**DATE:** January 2025  
**RE:** New OSS Project Assignment - compak (Compose Package Manager)

## Executive Summary

We're building an open-source package manager for Compose applications. Think `apt` or `npm`, but for multi-container applications. This addresses a significant gap in the container ecosystem where developers need something between raw Compose files and full Kubernetes deployments. Works with both Docker Compose and Podman Compose.

## Problem Statement

Currently, there's no standardized way to distribute, version, and install Compose applications. Developers resort to:
- Copy-pasting compose files from GitHub gists
- Building custom Makefile/bash script solutions
- Jumping to Kubernetes (overkill for many use cases)
- Using platform-specific solutions (lock-in)

Docker attempted this with Docker App (CNAB implementation) but deprecated it in 2021. The need remains unaddressed.

## Solution Overview

A lightweight CLI tool that enables:
```bash
compak install wordpress --version 6.1.0
compak list
compak uninstall wordpress
```

Packages are stored in OCI registries (Docker Hub, GHCR) as artifacts, leveraging existing infrastructure. Automatically detects and works with Docker Compose or Podman Compose.

## Technical Architecture

### Package Format
```
wordpress/
├── package.yaml         # Metadata + parameter definitions
├── docker-compose.yaml  # Service definitions with ${var} placeholders
└── values.yaml         # Default values (optional)
```

### Core Components

1. **CLI (Go)**
   - Commands: install, uninstall, list, publish, search
   - Minimal dependencies (cobra, oras, yaml parser)
   - Wraps detected compose tool (docker/podman)

2. **Compose Detection**
   - Auto-detects: Docker Compose v2, docker-compose, podman-compose
   - Falls back gracefully with clear error messages
   - Override via COMPOSE_COMMAND environment variable

3. **Registry Client**
   - OCI-compliant artifact storage
   - Use `oras` library for push/pull operations
   - Package URI: `ghcr.io/compak/wordpress:6.1.0`

4. **Template Engine**
   - Start with simple envsubst-style `${var}` replacement
   - No complex logic in v1 (avoid Helm's complexity)

5. **Local State Manager**
   - JSON file in `~/.compak/state.json`
   - Tracks installed packages, versions, and values

## Implementation Plan

### Phase 1: MVP (Week 1-2)
**Goal:** Working prototype that can install/uninstall a single package

```go
// Compose detection
func DetectComposeCommand() string {
    for _, cmd := range []string{
        "docker compose",
        "docker-compose", 
        "podman-compose",
    } {
        if commandExists(cmd) {
            return cmd
        }
    }
    return ""
}

// Core operations
func (c *Client) Install(pkg, version string, values map[string]string) error
func (c *Client) Uninstall(pkg string) error  
func (c *Client) List() ([]InstalledPackage, error)
```

**Deliverables:**
- [ ] Basic CLI with install/uninstall/list commands
- [ ] Docker/Podman detection logic
- [ ] OCI registry pull functionality
- [ ] Simple template substitution
- [ ] Local state tracking

### Phase 2: Publishing & Discovery (Week 3-4)
**Goal:** Authors can publish packages; users can discover them

- [ ] `compak publish` command
- [ ] `compak search` with basic registry API integration
- [ ] Documentation for package authors
- [ ] 3-5 example packages (WordPress, PostgreSQL, Redis)

### Phase 3: Polish & Release (Week 5-6)
**Goal:** Production-ready v0.1.0 release

- [ ] Error handling and recovery
- [ ] Integration tests for both Docker and Podman
- [ ] Installation scripts (brew, apt, binary releases)
- [ ] Documentation (README with clear examples)
- [ ] GitHub Actions CI/CD

## Technical Decisions

### Language: Go
- Single binary distribution
- Excellent container ecosystem integration
- Strong CLI libraries (cobra/viper)
- Cross-platform compilation

### Registry: OCI Artifacts
- Reuse existing infrastructure (Docker Hub, GHCR)
- No custom registry needed
- Established auth mechanisms

### Runtime Support: Docker + Podman
- Auto-detection of available tools
- Same package format works for both
- Expands potential user base significantly

### No Complex Features in v1
- No dependency resolution
- No hooks/lifecycle scripts  
- No multi-environment management
- No web UI

## Success Metrics

### Launch (Month 1)
- 100 GitHub stars
- 5 community-contributed packages
- Working on Docker and Podman
- macOS/Linux/Windows support

### Growth (Month 3)
- 1,000 GitHub stars
- 20+ packages in registry
- Active issue discussions
- 2-3 external contributors

### Validation (Month 6)
- 5,000+ stars
- Integration interest from Podman community
- Blog posts/tutorials from community
- Decision point: Continue investment or maintain as-is

## Risk Analysis

| Risk | Likelihood | Impact | Mitigation |
|------|------------|---------|------------|
| Low adoption | Medium | High | Start with personal use; success not required |
| Registry costs | Low | Low | Use free tiers; community hosting |
| Scope creep | High | Medium | Strict YAGNI policy for v1 |
| Compose format changes | Low | Medium | Pin to Compose spec v3.x initially |
| Name conflicts | Low | Low | "compak" appears unique in package manager space |

## Development Guidelines

### Code Style
- Keep it simple - no clever abstractions
- Extensive comments for OSS contributors
- Every public function has examples
- Error messages guide users to solutions

### Testing Strategy
- Unit tests for core logic
- Integration tests with both Docker and Podman
- Example packages as e2e tests
- Test on Linux/macOS/Windows

### Documentation First
- Write the README before the code
- User guide before features
- Clear package author documentation
- Document Docker/Podman compatibility

## Why You Should Be Excited

1. **Real Problem:** You've definitely felt this pain if you've used Compose
2. **Broader Reach:** Supporting Podman expands audience (RHEL/Fedora users)
3. **Greenfield:** Build it right from scratch
4. **Community Impact:** Helps both Docker and Podman ecosystems
5. **Resume Builder:** "Created cross-platform OSS tool with X stars"

## Your Autonomy

As the implementing engineer, you have full authority over:
- Implementation details
- Library choices
- Code structure
- Release timing

Non-negotiables:
- Keep v1 scope minimal
- Support both Docker and Podman from day 1
- Ship something working in 6 weeks

## Getting Started

1. Set up repo: `github.com/[yourname]/compak`
2. Create basic CLI structure with cobra
3. Implement compose detection logic
4. Test with both Docker and Podman environments
5. Ship v0.0.1-alpha for testing

## Questions to Consider

- Should we support Docker Compose v2 exclusively or maintain v1 compatibility?
- How do we handle conflicting network/volume names between packages?
- Should packages be installed globally or per-project?
- Do we need package signing from day 1?

Let's discuss these in our kickoff, but don't let them block initial development.

## Resources

- Docker App retrospective: https://github.com/docker/roadmap/issues/209
- CNAB spec (for context, not implementing): https://cnab.io
- OCI artifacts: https://github.com/opencontainers/artifacts
- Oras library: https://github.com/oras-project/oras-go
- Podman Compose: https://github.com/containers/podman-compose

---

**Next Steps:** 
1. Create GitHub repository
2. Set up basic CLI structure
3. Implement compose detection
4. Get `compak install` working with a local package
5. Test with both Docker and Podman

**Remember:** We're building a simple, useful tool that works for the entire Compose ecosystem. Ship early, iterate based on real usage.