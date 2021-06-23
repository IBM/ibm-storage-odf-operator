
## Files instruction

The main root directories list as below:

```
├── api
├── build
├── bundle
├── controllers
├── docs
├── config
├── hack
└── version
```

The directory `build` will keep the temporary files and it will be created during building. For instance, the executable binary files will be saved to `build/_output/bin`.

The directory `bundle` will keep the generated manifests and metadata files and it will be created while issuing the command `make bundle`. The file `bundle/metadata/dependencies.yaml` defines the required ibm-block-csi-operator minimum version.

The directory `hack` include all the bash scripts for Makefile. The script `common.sh` defines the common variables (e.g. versions, naming) which will be invoked by other scripts.

## Supported bundle building platforms

- macOS
- Linux which installed glibc 2.28+ (e.g. RHEL 8)

## Prerequisites on the building platforms

- Install the OpenShift CLI (oc) client version v4.7+
- Can connect to an OCP cluster specified in ~/.kube/config, run the command `oc login` to the cluster without problem.
- Run `docker login` to your registry successfully

## Building the internal catalog for OLM

### Building the operator image

There are few environment variables here, the file `hack/common.sh` defines default supported values, you can set the custom ones firstly then execute make commands.

```bash
$ export IMAGE_REGISTRY=<add new registry url here>
$ export REGISTRY_NAMESPACE=<add namespace here>
$ make docker-build
```

This example will build operator image and push it to the user defined registry. Current operator image will use version like "0.1.0-d300db3", it can be changed in `hack/common.sh` before GA.

### Building the bundle and internal catalog

You might set some environment variables before building the bundle, e.g. IBM Block CSI version or flashsystem driver image version.

```bash
$ export BLOCK_CSI_RELEASE=<add new version here>
$ export FLASHSYSTEM_DRIVER_RELEASE=<add new version here>
$ make bundle-build
$ make build-catalog
$ make deploy-catalog
```

By default, the above example will build a catalog image `ibm-storage-odf-catalog:latest` and add the catalog source to namespace `openshift-marketplace` on your OCP cluster. The bundle image will use version like "0.1.0", it can be changed in `hack/common.sh` before GA.

If only want to build the index image, issue the following commands then they will build the images then push to the registry.

```bash
$ make bundle-build
$ make build-catalog
```

To remove the catalog source on your OCP cluster, just issue the following command:

```bash
$ make undeploy-catalog
```

## References

### The environment variables list

You can set the following environment variables before issuing make commands.

```
OPM_VERSION
OPERATOR_SDK_VERSION
KUSTOMIZE_VERSION
BLOCK_CSI_RELEASE
BLOCK_CSI_CR_FILE
FLASHSYSTEM_DRIVER_RELEASE
OCS_OC_PATH
IMAGE_TAG
IMAGE_REGISTRY
REGISTRY_NAMESPACE
```
