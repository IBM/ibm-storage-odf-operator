#!/bin/bash

set -e

source hack/common.sh
source hack/ensure-opm.sh

echo "Creating an index image with the Operator bundle image injected..."
${OPM_BIN} -u docker -p docker index add --bundles "${BUNDLE_FULL_IMAGE_NAME}" --tag "${CATALOG_FULL_IMAGE_NAME}"

echo
echo "Pushing the index image to image registry..."
docker push "${CATALOG_FULL_IMAGE_NAME}"