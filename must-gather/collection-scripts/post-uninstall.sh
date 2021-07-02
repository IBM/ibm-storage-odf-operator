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


if [ -n "$(oc get storagecluster -n openshift-storage -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')" ]; then
    oc delete -f pod_helper.yaml
fi

# Add Ready nodes to the list
nodes=$(oc get nodes -l cluster.ocs.openshift.io/openshift-storage='' --no-headers | awk '/\yReady\y/{print $1}')

for node in ${nodes}; do
    pod_name=$(oc get pods -n openshift-storage | grep "${node//./}-debug" | awk '{print $1}')
    oc delete -n openshift-storage pod "$pod_name"
done
