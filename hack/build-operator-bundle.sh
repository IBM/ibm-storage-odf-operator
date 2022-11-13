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

if curl --head --silent --fail "${CSI_GA_CR_URL}" 2> /dev/null; then
  echo "CSI release is GAed. Using official images"
else
  echo "CSI tag doesn't exist yet, cloning CSI GitHub repository"
  oldPWD=$(pwd)
  if [ ! -d "${CSI_LOCAL_PATH}" ]
  then
      git clone "${CSI_GIT_PATH}"
      cd "${CSI_LOCAL_PATH}"
  else
      cd "${CSI_LOCAL_PATH}"
      git pull
  fi

  cd config/samples
  sed -i "s/ibmcom\/ibm-block-csi-driver-controller/${CSI_DEVELOP_REGISTRY}\/ibm-block-csi-driver-controller-amd64/g" "${CSI_CR_FILE}"
  sed -i "s/ibmcom\/ibm-block-csi-driver-node/${CSI_DEVELOP_REGISTRY}\/ibm-block-csi-driver-node-amd64/g" "${CSI_CR_FILE}"
  sed -i "s/tag: \"${CSI_RELEASE_NUMBER}\"/tag: \"latest\"/g" "${CSI_CR_FILE}"

  cd "${oldPWD}/${CSI_LOCAL_PATH}/${CSI_DOCKERFILE_PATH}"

  echo "Building and pushing CSI bundle image using ${CSI_DOCKERFILE_PATH}/${CSI_DOCKERFILE_NAME}"
  docker build -f "${CSI_DOCKERFILE_NAME}" -t "${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}" .
  docker tag "${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}" "${REGISTRY_NAMESPACE}/${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}"
  docker push "${REGISTRY_NAMESPACE}/${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}"

  echo "Deleting CSI repository clone"
  cd "${oldPWD}"
  rm -rf "${CSI_LOCAL_PATH}"
fi

