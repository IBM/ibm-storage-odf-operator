

# Overall design

## Summary

The Open Data Foundation (ODF) Driver for IBM Storage enables ODF monitoring
and managing IBM Storage systems. In north bound, this operator will implement the
interface with ODF. In south bound, it will manage IBM block CSI operator and
IBM Storage ODF Block driver. 

## Motivation

Currently, the Container Storage Interface (CSI) implemented by IBM block 
storage CSI driver can only provide limited function, e.g. provisioning,
snapshot. It's not convienent for OCP admin and app developer to get the
state of system, capacity, performance and so on.

The ODF is a well defined framework, which makes the third party storage
be the first citizen.

### Goals

1. Integrate installation/uninstallation/upgrade process with OLM.
2. Provide API, then ODF operator could create CR with nessary data got from
   Create Storage Cluster console.
3. Leverage OLM to install IBM block storage CSI operator.
4. Launch IBM Storage ODF driver when IBM Flashsystem storage cluster created.
5. Launch IBM block storage CSI driver when storage cluster created.
6. Create default StorageClass if pool name is provided by client in installation
   wizard. 
7. Dynamically pass this OCP using Flashsystem pool list to IBM storage ODF driver.

### Non-Goals

1. Suppport more than one Flashsystem.

## Proposal

None

## Design Details

Create Storage Cluster will triggle to create/update/delete a CR of IBM
Flashsystem Generic Info.

```yaml
apiVersion: odf.openshift.io/v1
kind: StorageSystemType
metadata:
  name: ibm-flashsystem
  namespace: openshift-storage
spec:
  operatorName: ibm-storage-operator
```

Once the `Secret` is changed, operator will restart the driver to utilize the
new `Secret`.

```yaml
kind: Secret
apiVersion: v1
metadata:
  name:  demo-secret
  namespace: default
type: Opaque
stringData:
  management_address: 192.168.9.10  
  username: superuser                
data:
  password: cGFzc3cwcmQ=             
```

This is API definition for IBM Storage ODF operator.

```yaml
apiVersion: odf.ibm.com/v1alpha1
kind: FlashSystemCluster
metadata:
  name: flashsystemcluster-sample
  namespace: ibm-storage-odf
spec:
  name: flashsystem-xxx
  secret:
    name: fs-secrets-example
    namespace: ibm-storage-odf
  insecureSkipVerify: true
  defaultPool:
    storageclassName: odf-sample
    poolName: Pool4
    fsType: ext4
    volumeNamePrefix: odf
    spaceEfficiency: thick
```

The operator will watch the StorageClass used by OCP. Once the provisioner of 
the StorageClass is `block.csi.ibm.com`, the corresponding pool will be updated
in this ConfigMap.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ibm-flashsystem-pools
  namespace: <according to operator installation>
data:
data:
  pools: '{"storageclass_pool":{"demo-storageclass":"Pool0","ibm-csi-mc80-storageclass":"Pool0","ibm-csi-mc80-storageclass-dedup-comp":"Pool0"}}'

```


In order to launch IBM block CSI driver, `ibmblockcsi` CR will be created by
operator.

### CSV

### Test Plan

TBD

### Monitoring Requirements

None

### Dependencies/limitations

1. This operator only support one instance only. So, ODF should prevent client from creating more if there is already a storage cluster created.

## Alternatives

None
