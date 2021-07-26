# Overall design

## Summary

The Open Data Foundation (ODF) Driver for IBM Storage enables ODF monitoring
and managing IBM Storage systems. In north bound, this operator will implement the
interface with ODF. In south bound, it will manage [IBM block CSI operator](https://github.com/IBM/ibm-block-csi-operator) and [IBM Storage ODF Block driver](https://github.com/IBM/ibm-storage-odf-block-driver).

## Motivation

Currently, the Container Storage Interface (CSI) implemented by IBM block
storage CSI driver can only provide limited functions, e.g. provisioning,
snapshot. It's not convienent for OCP admin and app developer to get the
state of system, capacity, performance and so on.

The ODF is a well defined framework, which makes the third party storage
be the first citizen.

### Goals

1. Integrate installation/uninstallation/upgrade process with OLM.
2. Provide API, then ODF operator could create CR with necessary data got from
   Create Storage Cluster console.
3. Leverage OLM to install IBM block storage CSI operator.
4. Launch IBM Storage ODF driver when IBM FlashSystem storage cluster created.
5. Launch IBM block storage CSI driver when storage cluster created.
6. Create default StorageClass for ODF over it.
7. Dynamically pass this OCP using FlashSystem pool list to IBM storage ODF driver.

### Dependencies/limitations

1. This operator only support one instance only. So, ODF should prevent client from creating more if there is already a FlashSystem cluster created.
