# ibm-storage-odf-operator

For RH OCS extension project, flashsystem operator component 


## build & install with OLM 

### 1. Prerequisites(test phase only)

1. Create new project with `openshift-*`. '

   ```
   oc adm new-project openshift-storage
   ```

2. Label the namespace

   ```bash
   oc label ns <your project name> "openshift.io/cluster-monitoring=true"
   ```

   Example

   ```bash
   oc label ns openshift-storage "openshift.io/cluster-monitoring=true"
   ```

### 2. Build & install (test phase only)

1. Clone build in your local repo. 

   ```bash
   git clone -b develop git@github.com:IBM/ibm-storage-odf-operator.git
   ```

2. Build bundle image

   ```bash
   export IMAGE_REGISTRY=<add new registry url here>
   export REGISTRY_NAMESPACE=<add namespace here>
   export FLASHSYSTEM_DRIVER_RELEASE=<add new version here>
   make bundle-build
   ```

### 3. Deploy ibm odf operator in operator hub

1. In openshift console, navigate to "Operators"->"OperatorHub" page, enter "odf" and filter, click "IBM Storage ODF operator" to install.
2. Choose the namespace you created in previous steps. 
3. Click "Install".

### 4. Create flashsystem cluster. 

1. In "Operators->Installed Operators" page, open "IBM Storage ODF operator" view.
2. Click "FlashSystem Cluster" tab, and click "Create Storage System".
3. Fill the fields here and click create button.  (This page is subjected to change.)


### 5. Uninstall

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

## best practice

1. In order to avoid the REST API token confliction, please assign a dedicated Flashsystem user to this operator.
