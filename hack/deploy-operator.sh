#!/bin/bash

set -e

source hack/common.sh
source hack/ensure-blockcsi-cryaml.sh

echo "Deploying the operator in the cluster..."

pushd config/manager
../../${KUSTOMIZE_BIN} edit set image controller="${OPERATOR_FULL_IMAGE_NAME}"
popd

${KUSTOMIZE_BIN} build config/default | oc apply -f -