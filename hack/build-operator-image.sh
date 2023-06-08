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

echo "Building Operator image and Pushing to image registry ${OPERATOR_FULL_IMAGE_NAME}..."
docker buildx build -t "${OPERATOR_FULL_IMAGE_NAME}" --build-arg VCS_REF=${VCS_REF} --build-arg VCS_URL=${VCS_URL} . --push
