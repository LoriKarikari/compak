<a id="readme-top"></a>

<br />
<div align="center">
  <h1 align="center">compak 🎒</h1>

  <p align="center">
    <a href="https://github.com/LoriKarikari/compak/actions/workflows/ci.yml">
      <img src="https://github.com/LoriKarikari/compak/actions/workflows/ci.yml/badge.svg" alt="CI">
    </a>
    <a href="https://github.com/LoriKarikari/compak/releases">
      <img src="https://img.shields.io/github/v/release/LoriKarikari/compak" alt="Release">
    </a>
    <a href="https://github.com/LoriKarikari/compak/blob/main/LICENSE">
      <img src="https://img.shields.io/github/license/LoriKarikari/compak" alt="License">
    </a>
    <a href="https://goreportcard.com/report/github.com/LoriKarikari/compak">
      <img src="https://goreportcard.com/badge/github.com/LoriKarikari/compak" alt="Go Report Card">
    </a>
  </p>

  <p align="center">
    A package manager for Docker Compose applications
    <!-- <br />
    <a href="https://github.com/LoriKarikari/compak"><strong>Explore the docs »</strong></a>
    <br /> -->
    <br />
    <a href="https://github.com/LoriKarikari/compak/issues/new?labels=bug">Report Bug</a>
    ·
    <a href="https://github.com/LoriKarikari/compak/issues/new?labels=enhancement">Request Feature</a>
  </p>
</div>


## About The Project

Compak allows you to install, manage, and distribute multi-container applications using Compose.

Inspired by Docker App ([deprecated in 2021](https://github.com/docker/roadmap/issues/209)), compak fills the gap for standardized distribution of Compose applications without the complexity of full orchestration platforms.


## Getting Started

### Prerequisites

* Docker Compose v2, docker-compose, or podman-compose
* Linux, macOS, or Windows

### Installation

1. Clone the repo
   ```bash
   git clone https://github.com/LoriKarikari/compak
   ```
2. Build the project
   ```bash
   cd compak
   make build
   ```
3. The binary will be available in `bin/compak`

## Usage

### Install a package from the index

```bash
compak install [package]
```

### Install from OCI registry

```bash
compak install ghcr.io/user/package:version
```

### Install with custom parameters

```bash
compak install [package] --set PORT=9090 --set DB_PASSWORD=secure
```

### Install from local directory

```bash
compak install mypackage --path ./my-package-dir
```

### List installed packages

```bash
compak list
```

### Check package status

```bash
compak status mypackage
```

### Uninstall a package

```bash
compak uninstall mypackage
```

### Search for packages

```bash
compak search [package]
```

### Update package index

```bash
compak update
```

## Package Format

Compak packages are defined in the package index (`paks/`) and reference upstream Compose files via URL.

Example package definition (`paks/myapp.yaml`):
```yaml
name: myapp
version: 1.0.0
description: Example application
author: maintainer-name
homepage: https://example.com
repository: https://github.com/example/myapp
source: https://raw.githubusercontent.com/example/myapp/main/docker-compose.yml

parameters:
  PORT:
    type: integer
    default: "8080"
    description: Port to expose application on
  DB_PASSWORD:
    type: string
    required: true
    description: Database password
```

When installed, compak downloads the compose file from the `source` URL and applies the configured parameters.

## Environment Variables

Compak supports the following environment variables for configuration:

* `COMPAK_INDEX_REPO` - Override the default package index repository (default: `https://github.com/LoriKarikari/compak.git`)
* `COMPAK_INDEX_PATH` - Override the package index subdirectory path (default: `paks`)
* `GITHUB_TOKEN` - GitHub personal access token for authenticating with GitHub Container Registry

### Testing with Local Package Index

For testing or development, you can point compak to a custom package index:

```bash
# Use test fixtures
COMPAK_INDEX_PATH=test/fixtures/paks compak search demo

# Use a different repository
COMPAK_INDEX_REPO=https://github.com/myorg/myindex.git compak update
```

## Contributing

### Test Fixtures

Compak includes test package fixtures in `test/fixtures/paks/` for testing the package index functionality. These demo packages follow the same pattern as Homebrew's test fixtures:

* `demo-wordpress.yaml` - Example WordPress setup
* `demo-nextcloud.yaml` - Example Nextcloud setup

To test with fixtures:

```bash
COMPAK_INDEX_PATH=test/fixtures/paks compak search demo
```

These fixtures are clearly labeled as demo packages and are not meant for production use.

## Acknowledgments

* [Docker App retrospective](https://github.com/docker/roadmap/issues/209) - Motivation for the project
* [OCI Artifacts specification](https://github.com/opencontainers/artifacts) - Foundation for package storage
* [oras-project](https://github.com/oras-project/oras-go) - OCI registry client library
* [Porter](https://github.com/getporter/porter) - Demonstrated patterns for package management
* [Homebrew](https://brew.sh) - Inspiration for package index and distribution model