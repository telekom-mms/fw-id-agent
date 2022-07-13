#!/bin/bash

GIT_TAG=$(git describe --tags --abbrev=0)
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_VERSION="$GIT_TAG-$GIT_COMMIT"

VERSION="github.com/T-Systems-MMS/fw-id-agent/internal/agent.version=$GIT_VERSION"

go build -ldflags "-X $VERSION" ./cmd/fw-id-agent
