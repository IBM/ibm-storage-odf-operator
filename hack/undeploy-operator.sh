#!/bin/bash

set -e

source hack/common.sh

echo "Undeploying the operator in the cluster..."

${KUSTOMIZE_BIN} build config/default | oc delete -f -