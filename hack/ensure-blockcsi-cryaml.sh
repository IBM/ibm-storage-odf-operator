#!/bin/bash

set -e

source hack/common.sh

CSI_CR_PATH="config/manager/${BLOCK_CSI_CR_FILE}"
CSI_CR_URL="https://raw.githubusercontent.com/IBM/ibm-block-csi-operator/${BLOCK_CSI_RELEASE}/deploy/crds/${BLOCK_CSI_CR_FILE}"

if [ ! -f "${CSI_CR_PATH}" ] || [[ -f "${CSI_CR_PATH}" && "$(grep release "${CSI_CR_PATH}" | awk -F ': ' '{print $2}')" != ${BLOCK_CSI_RELEASE} ]]; then
        echo "Downloading the IBM Block CSI CR file..."
        curl -JL "${CSI_CR_URL}" -o "${CSI_CR_PATH}"
else
        echo "Using the cached CR file ${BLOCK_CSI_CR_FILE}"
fi