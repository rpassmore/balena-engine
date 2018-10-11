// +build linux freebsd openbsd darwin

package client

// DefaultDockerHost defines os specific default if DOCKER_HOST is unset
const DefaultDockerHost = "unix:///var/run/balena-engine.sock"
