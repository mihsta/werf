#!/bin/bash

set -e

# FIXME(ilya-lesikov):
IMAGE_NAME=ilyalesikov/test:"$(git rev-parse HEAD)"
docker push $IMAGE_NAME
