#!/bin/bash

set -e

source hack/common.sh

echo "Installing the CRDs in the cluster..."

${KUSTOMIZE_BIN} build config/crd | oc apply -f -