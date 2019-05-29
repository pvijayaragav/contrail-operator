### Deploys Contrail using Operator 

# To start perform following steps

```
kubectl create -f crd.yaml -n kube-system
kubectl create -f operator.yaml -n kube-system
kubectl create -f roles.yaml -n kube-system
kubectl create -f km_roles.yaml -n kube-system
```
* operator image is pulled from pvijayaragav/operator_test:latest

```
kubectl create -f cr.yaml -n kube-system
```

# Finally label your nodes as infra
* you should label your contrail nodes which were provided in cr.yaml as infra like below
```
kubectl label node <> node-role.kubernetes.io/infra=true
```
