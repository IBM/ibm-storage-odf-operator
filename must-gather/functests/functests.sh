#!/usr/bin/env bash
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

if [ -z "$GOPATH" ]; then
    exit 1
fi

#shellcheck source=/dev/null
source "$GOPATH/src/github.com/openshift/ocs-operator/hack/common.sh"

# must-gather func-tests
must_gather_output_dir="$GOPATH/src/github.com/openshift/ocs-operator/must-gather-test"
# Cleaning all existing dump
rm -rf "${must_gather_output_dir}"
echo "Triggering ocs-must-gather"
${OCS_OC_PATH} adm must-gather --image="${MUST_GATHER_FULL_IMAGE_NAME}" --dest-dir="${must_gather_output_dir}"
"$GOPATH"/src/github.com/openshift/ocs-operator/must-gather/functests/integration/command_output_test.sh "${must_gather_output_dir}"

echo "All OCS-must-gather tests passed successfully"

# Cleaning the dump
rm -rf "${must_gather_output_dir}"
