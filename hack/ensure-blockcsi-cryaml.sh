#!/bin/bash
#
# Copyright contributors to the ibm-storage-odf-operator project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#


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
