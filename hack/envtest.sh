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

if [ ! -d "${OUTDIR}" ]; then
        mkdir -p "${OUTDIR}"
fi

OUTDIR_PATH="$(pwd)/${OUTDIR}"

test -f ${OUTDIR}/setup-envtest.sh || curl -sSLo ${OUTDIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
source ${OUTDIR}/setup-envtest.sh
fetch_envtest_tools ${OUTDIR}
setup_envtest_env ${OUTDIR_PATH}

# export TEST_USE_EXISTING_CLUSTER=true
#if [ ! -z $TEST_USE_EXISTING_CLUSTER ];then
    #kubectl apply -f config/samples/csi-crds/csi.ibm.com_ibmblockcsis_crd.yaml
    #if [ $? -ne 0 ];then
    #    exit 1
    #fi
#fi
export TEST_FS_CR_FILEPATH="$(pwd)/config/samples/csi.ibm.com_v1_ibmblockcsi_cr.yaml"
export TEST_FS_PROM_RULE_FILE="$(pwd)/rules/prometheus-flashsystem-rules.yaml"
go test -v ./... -coverprofile cover.out
