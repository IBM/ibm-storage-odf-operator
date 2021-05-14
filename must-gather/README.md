IBM Storage ODF Must-Gather
=================

`IBM Storage ODF Must-Gather` is a tool built on top of [OpenShift must-gather](https://github.com/openshift/must-gather)
that expands its capabilities to gather IBM Storage ODF information.

### Usage
```
oc adm must-gather --image=<ip address>/ibmcom/ibm-storage-odf-operator-must-gather -- gather [filter_string] <ibm-odf-namespace> <ibm-block-csi-namespace>
```

filter_string:
    Required. It is usually the string or substring of the name of cluster deployed by user.
    It is used with `grep` to filter out appropricate resourcesand objects.
ibm-odf-namespace:
    Optional. Specify this parameter when ODF is deployed in a non-default namespace.
ibm-block-csi-namespace:
    Optional. Specify this parameter when IBM block CSI is deployed in a non-default namespace.

The command above will create a local directory with a dump of the ODF state.
Note that this command will only get data related to the ODF part of the OpenShift cluster.

You will get a dump of:
- The ODF Operator (and its children objects)
- The IBM CSI Operator (and its children objects)
- Cluster-scoped resources

In order to get data about other parts of the cluster (not specific to ocs) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.

