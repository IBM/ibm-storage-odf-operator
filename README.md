# ibm-storage-odf-operator
For RH OCS extension project, flashsystem operator component 


## build & install

```bash

make build
make docker-build

make undeploy
make deploy

# apply configmap & secrets 
# TODO: generate configmap & secrets in operator based on fs CSI secret

kubectl apply -f config/samples/csi-secret.yaml
kubectl apply -f config/samples/flashsystem-configmap.yaml
kubectl apply -f config/samples/odf_v1alpha1_flashsystemcluster.yaml 

```
