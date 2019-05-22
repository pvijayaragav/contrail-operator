### Deploys Contrail using Operator 

# To start perform following steps

```
kubectl create -f crd.yaml -n kube-system
kubectl create -f operator.yaml -n kube-system
kubectl create -f roles.yaml -n kube-system
kubectl create -f km_roles.yaml -n kube-system
```
* operator image is pulled from pvijayaragav/operator_test:latest

# Edit the cr.yaml file
* Edit the cr.yaml file to provide the list of nodes where contrail wants to be run ( provide ',' seperated ips for HA )
* Also provide the kubernetes apiserver node ip
```
kubectl create -f cr.yaml -n kube-system
```

# Finally label your nodes as infra
* you should label your contrail nodes which were provided in cr.yaml as infra like below
```
kubectl label node <> node-role.kubernetes.io/infra=true
```
