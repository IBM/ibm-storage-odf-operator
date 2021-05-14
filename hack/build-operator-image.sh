#!/bin/bash

set -e

source hack/common.sh

echo "Building Operator image ${OPERATOR_FULL_IMAGE_NAME}..."
docker build -t "${OPERATOR_FULL_IMAGE_NAME}" --build-arg VCS_REF=${VCS_REF} --build-arg VCS_URL=${VCS_URL} .

echo
echo "Pushing Operator image to image registry..."
docker push "${OPERATOR_FULL_IMAGE_NAME}"