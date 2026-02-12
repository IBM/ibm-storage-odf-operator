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

#!/usr/bin/env bash
set -euo pipefail

source hack/common.sh

mkdir -p "${OUTDIR}"

# Install setup-envtest tool
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.23

# Download envtest binaries and get ABSOLUTE path
KUBEBUILDER_ASSETS_REL="$(setup-envtest use --bin-dir "${OUTDIR}" -p path 1.30.x)"
export KUBEBUILDER_ASSETS="$(cd "${KUBEBUILDER_ASSETS_REL}" && pwd)"

echo "KUBEBUILDER_ASSETS=${KUBEBUILDER_ASSETS}"

# Export test-specific variables
export TEST_FS_CR_FILEPATH="$(pwd)/config/samples/csi.ibm.com_v1_ibmblockcsi_cr.yaml"
export TEST_FS_PROM_RULE_FILE="$(pwd)/rules/prometheus-flashsystem-rules.yaml"

# Run tests
go test -v ./... -coverprofile cover.out
