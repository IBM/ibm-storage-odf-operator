## IBM Storage ODF (ISO) Operator

The Red Hat Open Data Foundation (ODF) Driver for IBM Storage enables ODF monitoring
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
$ export REGISTRY_NAMESPACE=<add namespace here>
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

#### Deploy ibm odf operator in OperatorHub

1. In openshift console, navigate to "Operators"->"OperatorHub" page, enter "odf" and filter, click "IBM Storage ODF operator" to install.
2. Choose the namespace you created in previous steps. 
3. Click "Install".

#### Create flashsystem cluster

1. In "Operators->Installed Operators" page, open "IBM Storage ODF operator" view.
2. Click "FlashSystem Cluster" tab, and click "Create Storage System".
3. Fill the fields here and click create button.  (This page is subjected to change.)

#### Uninstall

1. Delete the created custom resource of FlashSystemCluster.

   1. In "Operators->Installed Operators" page, open "IBM Storage ODF operator" view.
   2. Click "FlashSystem Cluster" tab.
   3. Select the "FlashSystemCluster" custom resource and click "Delete Storage System" from right kebab in the same line.

2. Delete "IBM Storage ODF operator".

   1. Locate to "Installed Operators" page and select the project which includes "IBM Storage ODF operator".
   2. Select the "IBM Storage ODF operator" and click "Uninstall Operator" from right kebab in the same line.

3. Delete "Operator for IBM block storage CSI driver" if it is running in same project of "IBM Storage ODF operator".

   **NOTES**:
   - If "IBM Storage ODF operator" is installed separately which is standalone without "IBM Storage ODF operator", skip step 3 & 4.
   - make sure there is no application/data consuming IBM block storage before removing the operator

   Reference: [IBM block CSI operator uninstallation](https://github.com/IBM/ibm-block-csi-operator#uninstalling)

   1. In "Operators->Installed Operators" page, open "Operator for IBM block storage CSI driver" view.
   2. Click "IBM block storage CSI driver" tab, select the "IBMBlockCSI" custom resource and click "Delete IBMBlockCSI" through right kebab in the same line.
   3. In "Operators->Installed Operators" page, select the project which includes "Operator for IBM block storage CSI driver".
   4. Select the "Operator for IBM block storage CSI driver" and click "Uninstall Operator" from right kebab in the same line.

4. Delete the project (namespace) which hold previous "IBM Storage ODF operator".

## Best Practice

In order to avoid the REST API token confliction, please assign a dedicated Flashsystem user to this operator.
