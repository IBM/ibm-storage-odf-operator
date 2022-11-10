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

echo "Building Operator bundle image ${BUNDLE_FULL_IMAGE_NAME}..."
docker build -f bundle.Dockerfile -t "${BUNDLE_FULL_IMAGE_NAME}" .

echo
echo "Pushing Operator bundle image to image registry..."
docker push "${BUNDLE_FULL_IMAGE_NAME}"

CSI_CR_URL="https://raw.githubusercontent.com/IBM/ibm-block-csi-operator/${BLOCK_CSI_RELEASE}/config/samples/${BLOCK_CSI_CR_FILE}"
if curl --head --silent --fail "${CSI_CR_URL}" 2> /dev/null; then
  echo "CSI release is GAed. Using official images"
else
  echo "CSI tag doesn't exist yet, cloning CSI GitHub repository"
  oldPWD=$(pwd)
  if [ ! -d "${DEFAULT_CSI_LOCAL_PATH}" ]
  then
      git clone "${DEFAULT_CSI_GIT_PATH}"
      cd "${DEFAULT_CSI_LOCAL_PATH}"
  else
      cd "${DEFAULT_CSI_LOCAL_PATH}"
      git pull
  fi

  cd "${DEFAULT_CSI_DOCKERFILE_PATH}"
  echo "Building and pushing CSI bundle image using ${DEFAULT_CSI_DOCKERFILE_PATH}/${DEFAULT_CSI_DOCKERFILE_NAME}"
  docker build -f "${DEFAULT_CSI_DOCKERFILE_NAME}" -t "${DEFAULT_CSI_DEVELOP_IMAGE}:${IMAGE_TAG}" .
  docker tag "${DEFAULT_CSI_DEVELOP_IMAGE}:${IMAGE_TAG}" "${REGISTRY_NAMESPACE}/${DEFAULT_CSI_DEVELOP_IMAGE}:${IMAGE_TAG}"
  docker push "${REGISTRY_NAMESPACE}/${DEFAULT_CSI_DEVELOP_IMAGE}:${IMAGE_TAG}"

  echo "Deleting CSI repository clone"
  cd "${oldPWD}"
  rm -rf "${DEFAULT_CSI_LOCAL_PATH}"
fi

