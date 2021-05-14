#!/bin/bash

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