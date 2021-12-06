#!/bin/bash

set -e

BUILD_VERSION="$(git rev-parse HEAD)"
# FIXME(ilya-lesikov):
IMAGE_NAME=ilyalesikov/test:$BUILD_VERSION
docker build -f scripts/werf-in-docker/Dockerfile --build-arg build_version=$BUILD_VERSION -t $IMAGE_NAME .
