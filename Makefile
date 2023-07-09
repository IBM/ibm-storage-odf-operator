export GOPROXY=https://proxy.golang.org
export IMAGE_REGISTRY=quay.io/ben_menachem
export IMAGE_TAG=1.5.0
export ENABLE_UPGRADE=True
CONTROLLER_GEN_VERSION="v0.4.1"

CSV_PATH=bundle/manifests/ibm-storage-odf-operator.clusterserviceversion.yaml

CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

LINT_VERSION="1.40.0"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

ensure-yq:
	@echo "Ensuring yq CLI tool installed..."
	hack/ensure-yq.sh

ensure-opm:
	@echo "Ensuring opm CLI tool installed..."
	hack/ensure-opm.sh

ensure-operator-sdk:
	@echo "Ensuring operator-sdk CLI tool installed..."
	hack/ensure-operator-sdk.sh

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

test: manifests generate fmt vet ## Run tests
	hack/envtest.sh

add-copyright:
	hack/add-copyright.sh

check-copyright:
	hack/check-copyright.sh

.PHONY: deps
deps:
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint --version)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: deps
	golangci-lint run -E gosec --timeout=6m    # Run `make lint-fix` may help to fix lint issues.

.PHONY: lint-fix
lint-fix: deps	
	golangci-lint run --fix

##@ Build

build: generate fmt vet ## Build manager binary
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host
	go run ./main.go

docker-build:  ## Build docker image with the manager
	hack/build-operator-image.sh

.PHONY: bundle ## Generate bundle manifests and metadata, then validate generated files.
bundle: ensure-operator-sdk manifests kustomize ensure-yq
	hack/bundle-manifests.sh

bundle-build: bundle ## Build the bundle image
	hack/build-operator-bundle.sh

build-catalog:
	hack/build-operator-catalog.sh

build-must-gather:
	hack/build-must-gather.sh

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config
	hack/install-crds.sh

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config
	hack/uninstall-crds.sh

deploy-catalog:
	hack/deploy-catalog-source.sh

undeploy-catalog:
	hack/delete-catalog-source.sh

deploy: manifests kustomize csicryaml ## Deploy controller to the K8s cluster specified in ~/.kube/config
	hack/deploy-operator.sh

undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config
	hack/undeploy-operator.sh

.PHONY: clean
clean:
	hack/clean.sh

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION})

kustomize: ## Download kustomize locally if necessary
	hack/ensure-kustomize.sh

csicryaml:
	hack/ensure-blockcsi-cryaml.sh

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
