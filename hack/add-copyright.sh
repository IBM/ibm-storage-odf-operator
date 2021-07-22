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
PROTECT_ROOT=$(cd $CURRENT_PATH/..; pwd)
BOILERPLATE_GO=$CURRENT_PATH/boilerplate.go.txt
BOILERPLATE_SH=$CURRENT_PATH/boilerplate.sh.txt

LOCAL_OS_TYPE=$(uname)

ibm_cr="Copyright contributors to the .* project"
apache_cr="Apache License, Version 2"

skip_n=0
add_n=0

mac=0
if [ "$LOCAL_OS_TYPE" == "Darwin" ] && [[ "$(sed --version 2>&1 | head -n 1 | awk -F " " '{print $2}')" == "illegal" ]]; then
  mac=1
fi

#ignore directory vendor, .git, node_modules
#check file go, proto, sh, js, ts

for file in $(find $PROTECT_ROOT -not -path "*/vendor/*" -not -path "*/.git/*" -not -path "*/node_modules/*" -type f \( -name '*.go' -o -name '*.proto' -o -name '*.sh' -o -name '*.js' -o -name '*.ts' \)); do
  if [ ! -s $file ]; then
    #echo "$file is empty, skip"
    let skip_n+=1
    continue
  fi

  if [[ $(grep -m 1 "$apache_cr" "$file") ]] && [[ $(grep -m 1 "$ibm_cr" "$file") ]] 
  then
    # the file already has copyright.
    # echo "-- skip: $file already has copyright"
    let skip_n+=1
  else
    let add_n+=1
    # add different comment format to shell with '#' and other files with '/* */'
    echo "$add_n> add copyright header to $file "
    if [[ $file == *.sh ]]
    then 
      #echo "this is a shell file"
      firstline=$(sed -n 1p $file)
      #echo $firstline
      if [[ $firstline == \#!* ]] 
      then
        #echo "first line has #!"
        if [ $mac -eq 1 ]; then
          sed -i '' "1r $BOILERPLATE_SH" $file
        else
          sed -i "1r $BOILERPLATE_SH" $file
        fi
      else
        if [ $mac -eq 1 ]; then
          sed -i '' '1i\
          #
          ' $file
          sed -i '' "1r $BOILERPLATE_SH" $file
          sed -i '' '1d' $file
        else
          sed -i '1i #' $file
          sed -i "1r $BOILERPLATE_SH" $file
          sed -i '1d' $file
        fi
      fi
      
    else
      #echo "this is not a shell file"
      if [ $mac -eq 1 ]; then
        sed -i '' '1i\
        //
        ' $file
        sed -i '' "1r $BOILERPLATE_GO" $file
        sed -i '' '1d' $file
      else
        sed -i '1i //' $file
        sed -i "1r $BOILERPLATE_GO" $file
        sed -i '1d' $file
      fi
    fi
  fi
done

echo "" 
echo "Summary:"
echo "    $skip_n files already have copyright or are empty"
echo "    $add_n files were added with copyright"
echo ""



