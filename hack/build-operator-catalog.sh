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
source hack/ensure-opm.sh

CSI_BUNDLE_IMAGE=""
if curl --head --silent --fail "${CSI_GA_CR_URL}" &> /dev/null; then
  echo "CSI release is GAed. Using official images"
else
  echo "CSI tag doesn't exist yet, adding CSI bundle into ODF internal catalog."
  CSI_BUNDLE_IMAGE="${IMAGE_REGISTRY}/${REGISTRY_NAMESPACE}/${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}"
fi

echo "Creating an index image with the Operator bundle image injected..."
if [ -z "${CSI_BUNDLE_IMAGE}" ]; then
  ${OPM_BIN} -u docker -p docker index add --bundles "${BUNDLE_FULL_IMAGE_NAME}" --tag "${CATALOG_FULL_IMAGE_NAME}"
else
  ${OPM_BIN} -u docker -p docker index add --bundles "${BUNDLE_FULL_IMAGE_NAME},${CSI_BUNDLE_IMAGE}" --tag "${CATALOG_FULL_IMAGE_NAME}"
fi

echo
echo "Pushing the index image to image registry..."
docker push "${CATALOG_FULL_IMAGE_NAME}"
