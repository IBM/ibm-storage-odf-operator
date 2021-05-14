#!/bin/bash

set -e

source hack/common.sh

if [ ! -d "${OUTDIR}" ]; then
    mkdir -p "${OUTDIR}"
fi

OUTDIR_PATH="$(pwd)/${OUTDIR}"

test -f ${OUTDIR}/setup-envtest.sh || curl -sSLo ${OUTDIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
source ${OUTDIR}/setup-envtest.sh
fetch_envtest_tools ${OUTDIR}
setup_envtest_env ${OUTDIR_PATH}

export TEST_USE_EXISTING_CLUSTER=true 
go test ./... -coverprofile cover.out