## IBM Storage ODF (ISO) Operator

The Red Hat Open Data Foundation (ODF) Plugin for IBM Storage enables ODF monitoring
and managing IBM FlashSystem storage.

This is the official operator to deploy and manage IBM FlashSystem storage.

## Development

### Tools

- [Operator SDK](https://github.com/operator-framework/operator-sdk)

- [Kustomize](https://github.com/kubernetes-sigs/kustomize)

- [controller-gen](https://github.com/kubernetes-sigs/controller-tools)

### Build

#### ISO Operator

The operator image can be built via:

```
$ make docker-build
```

#### ISO Operator Bundle

To create an operator bundle image, run

```
$ make bundle-build
```

> Note: Push the Bundle image to image registry before moving to next step.

#### ISO Operator Index

An operator index image can be built using

```
$ make build-catalog
```

### Deploying development builds

#### Prerequisites(test phase only)

Create new project with `openshift-*`

```
$ oc adm new-project openshift-storage
```

Label the namespace

```bash
$ oc label ns <your project name> "openshift.io/cluster-monitoring=true"
```

Example:

```bash
$ oc label ns openshift-storage "openshift.io/cluster-monitoring=true"
```

#### Build the operator/bundle/index images

To install own development builds of ISO, first build and push the operator image to your own image repository.

```
$ export IMAGE_REGISTRY=<add new registry url here>
$ export IMAGE_TAG=<some-tag>
$ make docker-build
```

Once the operator image is pushed, build and push the operator bundle image.

```
$ make bundle-build
```

Next build and push the operator index image.

```
$ make build-catalog
```

Now add a catalog source to your OCP cluster.

```
$ make deploy-catalog
```

Then you can install the operator from OperatorHub on OCP cluster.

#### Deploy ibm storage odf operator

As current design, ibm storage odf operator is hidden on OperatorHub, and will be deployed by Red Hat OpenShift Data Foundation operator.

So you should deploy Red Hat OpenShift Data Foundation operator firstly, follow up its instructions from [here](https://github.com/red-hat-storage/odf-operator) or install the ODF GA version from OperatorHub.

Later, customize odf-operator via editing configmap to use correct catalog source which have ibm storage odf operator.

```
$ oc -n openshift-storage edit configmap odf-operator-manager-config
apiVersion: v1
data:
  IBM_SUBSCRIPTION_CATALOGSOURCE: ibm-storage-odf-catalog
  IBM_SUBSCRIPTION_CATALOGSOURCE_NAMESPACE: openshift-marketplace
```

Restart the operator odf-operator-controller-manager to ensure new values take effect.

```
$ oc -n openshift-storage get pod | grep odf-operator-controller-manager
$ oc -n openshift-storage delete pod odf-operator-controller-manager-xxx
```

In "Operators->Installed Operators" page, open "OpenShift Data Foundation" view.
Click "Storage Systems" tab, and click "Create StorageSystem".

Choose "Connect an external storage platform", select "IBM FlashSystem Storage" and click "Next".

Fill the required fields on the remaining window, at last click "Create StorageSystem".

#### Uninstall

1. Delete the created custom resources of ODF Storage Systems.

   1. In "Operators->Installed Operators" page, open "OpenShift Data Foundation" view.
   2. Click "Storage Systems" tab.
   3. Select the "StorageSystems" custom resources and click "Delete StorageSystem" action.

2. Delete "Operator for IBM block storage CSI driver" if it is running in same project of "IBM Storage ODF operator".

   **NOTES**:
   - If the "Operator for IBM block storage CSI driver" is installed separately which is standalone without "IBM Storage ODF operator", skip step 2 & 3.
   - Make sure there is no application/data consuming IBM block storage before removing the operator.

   Reference: [IBM block CSI operator uninstallation](https://github.com/IBM/ibm-block-csi-operator#uninstalling)

   1. In "Operators->Installed Operators" page, open "Operator for IBM block storage CSI driver" view.
   2. Click "IBM block storage CSI driver" tab, select the "IBMBlockCSI" custom resource and click "Delete IBMBlockCSI" through right kebab in the same line.
   3. In "Operators->Installed Operators" page, select the project which includes "Operator for IBM block storage CSI driver".
   4. Select the "Operator for IBM block storage CSI driver" and click "Uninstall Operator" from right kebab in the same line.

3. Delete the project (namespace) which hold previous "IBM Storage ODF operator".

## Running Unit test

Unit tests can be run via

```
make test
```

## Best Practice

In order to avoid the REST API token confliction, please assign a dedicated Flashsystem user to this operator.
