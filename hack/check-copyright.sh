#!/bin/bash -e
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

# add the copyright header for all code files using the boilerplate file

CURRENT_PATH=$(dirname "$BASH_SOURCE")
#echo "current path:" $CURRENT_PATH

PROTECT_ROOT=$(cd $CURRENT_PATH/..; pwd)
#echo "project path:" $PROTECT_ROOT

LOCAL_OS_TYPE=$(uname)

ibm_cr="Copyright contributors to the .* project"
apache_cr="Apache License, Version 2"

pass=0
fail=0

#ignore directory vendor, .git, node_modules
#check file go, proto, sh, js, ts

for file in $(find $PROTECT_ROOT -not -path "*/vendor/*" -not -path "*/.git/*" -not -path "*/node_modules/*" -type f \( -name '*.go' -o -name '*.proto' -o -name '*.sh' -o -name '*.js' -o -name '*.ts' \)); do
  if [ ! -s $file ]; then
    #echo "$file is empty, skip"
    let pass+=1
    continue
  fi

  if [[ $(grep -m 1 "$apache_cr" "$file") ]] && [[ $(grep -m 1 "$ibm_cr" "$file") ]]
  then
    # the file already has copyright.
    # echo "pass: $file already has copyright"
    let pass+=1
  else
    let fail+=1
    echo "No correct copyright info: $file"
  fi
done

echo "" 
echo "Check copyright summary:"
#echo "    $pass files pass copyright check."
if [[ $fail > 0 ]]; then
    echo "    $fail files fail copyright check. Run 'make add-copyright' to add copyright."
    exit 1
else
    echo "    All files pass copyright check."
fi
echo ""
