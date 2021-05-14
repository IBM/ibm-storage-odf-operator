#!/bin/bash

set -e

source hack/common.sh

echo "Uninstalling the CRDs in the cluster..."

${KUSTOMIZE_BIN} build config/crd | oc delete -f -