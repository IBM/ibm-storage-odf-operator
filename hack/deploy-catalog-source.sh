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
source hack/delete-catalog-source.sh

cat <<EOF | tee >(oc apply -f -) | cat
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${CATALOG_IMAGE_NAME}
  namespace: openshift-marketplace
spec:
  displayName: ${CATALOG_IMAGE_NAME}
  publisher: IBM Content
  sourceType: grpc
  image: ${CATALOG_FULL_IMAGE_NAME}
  updateStrategy:
    registryPoll:
      interval: 45m
EOF
