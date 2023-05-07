#!/bin/bash

GIT_TAG=$(git describe --tags --abbrev=0)
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_VERSION="$GIT_TAG-$GIT_COMMIT"

GO_MODULE="github.com/T-Systems-MMS/fw-id-agent"
VERSION="$GO_MODULE/internal/agent.Version=$GIT_VERSION"

go build -ldflags "-X $VERSION" ./cmd/fw-id-agent
go build -ldflags "-X $VERSION" ./cmd/fw-id-cli
