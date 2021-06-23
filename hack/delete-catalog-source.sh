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

if [[ "$(oc -n openshift-marketplace get catalogsource "${CATALOG_IMAGE_NAME}" 2>&1 | head -n 1 | awk -F ':' '{print $1}')" != "Error from server (NotFound)" ]]; then
        echo "Will remove the existing catalogsource ${CATALOG_IMAGE_NAME}..."
        oc -n openshift-marketplace delete catalogsource "${CATALOG_IMAGE_NAME}"
fi
