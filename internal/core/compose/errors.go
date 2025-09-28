package compose

import "errors"

var (
	ErrNoComposeFound = errors.New("no compose command found: please install Docker Compose, docker-compose, or podman-compose")
)
