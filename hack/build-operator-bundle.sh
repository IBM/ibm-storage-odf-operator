#!/bin/bash

set -e

source hack/common.sh

echo "Building Operator bundle image ${BUNDLE_FULL_IMAGE_NAME}..."
docker build -f bundle.Dockerfile -t "${BUNDLE_FULL_IMAGE_NAME}" .

echo
echo "Pushing Operator bundle image to image registry..."
docker push "${BUNDLE_FULL_IMAGE_NAME}"