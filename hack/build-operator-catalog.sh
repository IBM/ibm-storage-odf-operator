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
  CSI_BUNDLE_IMAGE="${IMAGE_REGISTRY}/${CSI_DEVELOP_BUNDLE_FULL_IMAGE_NAME}:${IMAGE_TAG}"
fi

#echo "Creating an index image with the Operator bundle image injected..."
#if [ -z "${CSI_BUNDLE_IMAGE}" ]; then
#  ${OPM_BIN} -u docker -p docker index add --bundles "${BUNDLE_FULL_IMAGE_NAME}" --tag "${CATALOG_FULL_IMAGE_NAME}"
#else
#  ${OPM_BIN} -u docker -p docker index add --bundles "${BUNDLE_FULL_IMAGE_NAME},${CSI_BUNDLE_IMAGE}" --tag "${CATALOG_FULL_IMAGE_NAME}"
#fi

#echo
#echo "Pushing the index image to image registry..."
#docker push "${CATALOG_FULL_IMAGE_NAME}"


build_push_file_based_catalog_quay_io_image() {
  catalog_package_name="${CATALOG_IMAGE_NAME}"
  operator_package_name="${OPERATOR_IMAGE_NAME}"
  channel="${CHANNELS}"
  bundle_quay_io_image="${BUNDLE_FULL_IMAGE_NAME}"
  operator_package_name_version="${OPERATOR_IMAGE_NAME_VERSION}"
  catalog_quay_io_image="${CATALOG_FULL_IMAGE_NAME}"
  initialize_catalog "${catalog_package_name}" "${channel}" "${operator_package_name}"
  add_bundle_to_catalog "${catalog_package_name}" "${bundle_quay_io_image}"
  add_channel_entry_for_bundle "${catalog_package_name}" "${channel}" "${operator_package_name}" "${operator_package_name_version}"

  ${OPM_BIN} validate "${catalog_package_name}"
  build_push_catalog_image "${catalog_package_name}" "${catalog_quay_io_image}"
}

initialize_catalog() {
  catalog_package_name=${1}
  channel=${2}
  operator_package_name=${3}
  mkdir "${catalog_package_name}" || exit
  echo "Generating catalog Dockerfile"
  ${OPM_BIN} alpha generate dockerfile "${catalog_package_name}"
  echo "Initializing catalog"
  ${OPM_BIN} init "${operator_package_name}" --default-channel="${channel}" --output yaml > "${catalog_package_name}"/index.yaml
}

add_bundle_to_catalog() {
  catalog_package_name=${1}
  bundle_full_image_name=${2}
  echo "Add bundle image into catalog"
  ${OPM_BIN} render "${bundle_full_image_name}" --output=yaml >> "${catalog_package_name}"/index.yaml
}

add_channel_entry_for_bundle() {
  catalog_package_name=${1}
  channel=${2}
  operator_package_name=${3}
  operator_package_name_version=${4}
  echo "Adding operator channel entry into catalog"
  cat << EOF >> "${catalog_package_name}"/index.yaml
---
schema: olm.channel
package: ${operator_package_name}
name: ${channel}
entries:
- name: ${operator_package_name_version}
EOF
}

build_push_catalog_image() {
  catalog_package_name=${1}
  catalog_quay_io_image=${2}
  echo "Building and pushing catalog"
  docker build -f "${catalog_package_name}".Dockerfile -t "${catalog_quay_io_image}" .
  docker push "${catalog_quay_io_image}"
}

build_push_file_based_catalog_quay_io_image
echo "Cleaning leftovers"
rm -rf "${CATALOG_IMAGE_NAME}".Dockerfile "${CATALOG_IMAGE_NAME}"