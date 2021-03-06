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

echo "Deploying the operator in the cluster..."

pushd config/manager
if [ "$LOCAL_OS_TYPE" == "Darwin" ] && [[ "$(sed --version 2>&1 | head -n 1 | awk -F " " '{print $2}')" == "illegal" ]]; then
        sed -i "" "s#value: .*#value: ${FLASHSYSTEM_DRIVER_FULL_IMAGE_NAME}#" ../default/manager_config_patch.yaml
else
        sed -i "s#value: .*#value: ${FLASHSYSTEM_DRIVER_FULL_IMAGE_NAME}#" ../default/manager_config_patch.yaml
fi

../../${KUSTOMIZE_BIN} edit set image controller="${OPERATOR_FULL_IMAGE_NAME}"
popd

${KUSTOMIZE_BIN} build config/default | oc apply -f -
