<a id="readme-top"></a>

<br />
<div align="center">
  <h3 align="center">compak</h3>

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

compak allows you to install, manage, and distribute multi-container applications using Docker Compose. Packages are stored in OCI registries like GitHub Container Registry or Docker Hub.  
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

### Install a package from a registry

```bash
compak install ghcr.io/user/package:version
```

### Install with custom parameters

```bash
compak install ghcr.io/user/package:version --set PORT=9090 --set SERVER_NAME=myserver
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

## Package Format

A compak package consists of two files:

* `package.yaml` - Package metadata and parameter definitions
* `docker-compose.yaml` - Service definitions with variable placeholders

Example `package.yaml`:
```yaml
name: nginx
version: 1.0.0
description: Simple nginx web server
parameters:
  PORT:
    type: port
    default: "8080"
    description: Port to expose nginx on
    required: false
```

Example `docker-compose.yaml`:
```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "${PORT}:80"
```

## Publishing Packages

Use the oras CLI to publish packages to OCI registries:

```bash
oras push ghcr.io/user/package:version \
  --artifact-type application/vnd.compak.package.v1+tar \
  package.yaml:application/vnd.compak.package.config.v1+yaml \
  docker-compose.yaml:application/vnd.compak.compose.v1+yaml
```

## Roadmap

- [x] Basic CLI with install/uninstall/list commands
- [x] Docker/Podman Compose detection
- [x] OCI registry pull functionality
- [x] Parameter substitution
- [x] Local state tracking
- [ ] Package publishing command
- [ ] Package search functionality
- [ ] Installation scripts (brew, apt, binary releases)

## Acknowledgments

* [Docker App retrospective](https://github.com/docker/roadmap/issues/209) - Inspiration for this solution
* [OCI Artifacts specification](https://github.com/opencontainers/artifacts) - Foundation for package storage
* [oras-project](https://github.com/oras-project/oras-go) - OCI registry client library
