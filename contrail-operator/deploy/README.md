### Deploys Contrail using Operator 

# To start perform following steps

```
kubectl create -f crd.yaml -n kube-system
kubectl create -f operator.yaml -n kube-system
kubectl create -f roles.yaml -n kube-system
kubectl create -f km_roles.yaml -n kube-system
kubectl create -f cr.yaml -n kube-system
```

# all contrail components run on master node now
