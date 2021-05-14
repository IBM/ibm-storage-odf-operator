#!/bin/bash

set -e

source hack/common.sh

if [ "$LOCAL_OS_TYPE" == "Darwin" ]; then
        YQ_PLATFORM=darwin_amd64
fi

YQ_URL="https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_${YQ_PLATFORM}"

if [ ! -d "${OUTDIR_BIN}" ]; then
    mkdir -p "${OUTDIR_BIN}"
fi

if [ ! -x "${YQ_BIN}" ] || [[ -x "${YQ_BIN}" && "$(${YQ_BIN} --version | awk -F ' ' '{print $3}')" != "${YQ_VERSION}" ]]; then
        echo "Downloading yq v${YQ_VERSION} CLI tool for ${LOCAL_OS_TYPE}..."
        curl -JL "${YQ_URL}" -o "${YQ_BIN}"
        chmod +x "${YQ_BIN}"
else
        echo "Using yq cached at ${YQ_BIN}"
fi