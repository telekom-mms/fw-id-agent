#!/bin/bash

GIT_TAG=$(git describe --tags --abbrev=0)
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_VERSION="$GIT_TAG-$GIT_COMMIT"

GO_MODULE="github.com/T-Systems-MMS/fw-id-agent"
AGENT_VERSION="$GO_MODULE/internal/agent.version=$GIT_VERSION"
CLI_VERSION="$GO_MODULE/internal/cli.version=$GIT_VERSION"

go build -ldflags "-X $AGENT_VERSION" ./cmd/fw-id-agent
go build -ldflags "-X $CLI_VERSION" ./cmd/fw-id-cli
