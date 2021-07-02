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


# This test checks for empty output files of ceph commands

# Expect base collection path as an argument
BASE_PATH=$1

# Use PWD as base path if no argument is passed
if [ "${BASE_PATH}" = "" ]; then
    BASE_PATH=$(pwd)
fi

# finding the paths of ceph command output directories
# shellcheck disable=SC2044
for path in $(find "${BASE_PATH}" -type d -name must_gather_commands); do
    numberOfEmptyOutputFiles=$(find "${path}" -empty -type f | wc -l)
    if [ "${numberOfEmptyOutputFiles}" -ne "0" ]; then
        printf "The following files must not be empty : \n\n";
        find "${path}" -empty -type f 
        exit 1;
    fi

    jsonOutputPath=${path}/json_output
    numberOfEmptyOutputFiles=$(find "${jsonOutputPath}" -empty -type f | wc -l)
    if [ "${numberOfEmptyOutputFiles}" -ne "0" ]; then
        printf "The following files must not be empty : \n\n";
        find "${jsonOutputPath}" -empty -type f 
        exit 1;
    fi
done

