#!/bin/bash

set -e

IMAGE_NAME=ghcr.io/werf/werf:"$(git rev-parse HEAD)"
docker push $IMAGE_NAME