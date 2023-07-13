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


export RESOURCES_NAMESPACE=openshift-storage
export EXPORTER_IMAGE=docker.io/ibmcom/ibm-storage-odf-block-driver:v0.0.22
export OPERATOR_POD_NAME=ibm-storage-odf-operator
export TEST_FS_CR_FILEPATH="$(pwd)/config/samples/csi.ibm.com_v1_ibmblockcsi_cr.yaml"
export TEST_FS_PROM_RULE_FILE="$(pwd)/rules/prometheus-flashsystem-rules.yaml"

exec bin/manager
