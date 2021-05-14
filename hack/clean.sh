#!/bin/bash

set -e

source hack/common.sh

echo "Cleaning previous outputs if have..."
if [ -d "${OUTDIR}" ]; then
    rm -rf "${OUTDIR}"
else
    echo "The previous outputs cleaned."
fi