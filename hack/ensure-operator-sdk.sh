#!/bin/bash

set -e

source hack/common.sh

if [ "$LOCAL_OS_TYPE" == "Darwin" ]; then
        OPERATOR_SDK_PLATFORM=darwin_amd64
fi

OPERATOR_SDK_URL="https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_${OPERATOR_SDK_PLATFORM}"

if [ ! -d "${OUTDIR_BIN}" ]; then
    mkdir -p "${OUTDIR_BIN}"
fi

if [ ! -x "${OPERATOR_SDK_BIN}" ] || [[ -x "${OPERATOR_SDK_BIN}" && "$(${OPERATOR_SDK_BIN} version | awk -F '"' '{print $2}')" != "${OPERATOR_SDK_VERSION}" ]]; then
        echo "Downloading operator-sdk ${OPERATOR_SDK_VERSION} CLI tool for ${LOCAL_OS_TYPE}..."
        curl -JL "${OPERATOR_SDK_URL}" -o "${OPERATOR_SDK_BIN}"
        chmod +x "${OPERATOR_SDK_BIN}"
else
        echo "Using operator-sdk cached at ${OPERATOR_SDK_BIN}"
fi