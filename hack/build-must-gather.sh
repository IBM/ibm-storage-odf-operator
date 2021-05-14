#!/bin/bash

set -e

source hack/common.sh

docker build -f must-gather/Dockerfile -t "${MUST_GATHER_FULL_IMAGE_NAME}" must-gather/
docker push "${MUST_GATHER_FULL_IMAGE_NAME}"
