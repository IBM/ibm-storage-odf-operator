#!/bin/bash

set -e

source hack/common.sh

if [ "$LOCAL_OS_TYPE" == "Darwin" ]; then
        OPM_PLATFORM=darwin-amd64-opm
fi

OPM_URL="https://github.com/operator-framework/operator-registry/releases/download/${OPM_VERSION}/${OPM_PLATFORM}"

if [ ! -d "${OUTDIR_BIN}" ]; then
    mkdir -p "${OUTDIR_BIN}"
fi

if [ ! -x "${OPM_BIN}" ] || [[ -x "${OPM_BIN}" && "$(${OPM_BIN} version | awk -F '"' '{print $2}')" != "${OPM_VERSION}" ]]; then
        echo "Downloading opm ${OPM_VERSION} CLI tool for ${LOCAL_OS_TYPE}..."
        curl -JL "${OPM_URL}" -o "${OPM_BIN}"
        chmod +x "${OPM_BIN}"
else
        echo "Using opm cached at ${OPM_BIN}"
fi