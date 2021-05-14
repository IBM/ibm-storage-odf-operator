#!/bin/bash

set -e

source hack/common.sh

oc -n openshift-marketplace delete catalogsource "${CATALOG_IMAGE_NAME}"