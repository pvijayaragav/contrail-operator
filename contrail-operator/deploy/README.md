### Deploys Contrail using Operator kubernetes

# To start perform following steps

```
kubectl create -f crd.yaml -n kube-system
kubectl create -f operator.yaml -n kube-system
kubectl create -f roles.yaml -n kube-system
kubectl create -f km_roles.yaml -n kube-system
kubectl create -f cr.yaml -n kube-system
```

### Deploys Contrail using Operator kubernetes

# Installation using openshift binary

```
./openshift_installer create install-config
```
1. now change the OpenshiftSDN parameter to contrail, and make masters node to 1 instead of 3

```
./openshift_installer create manifests
```

2. Above step creates manifests.
3. Copy the files above to manifests directory and name the file contrail- , so that
they are placed correctly

```
./openshift_installer create cluster
```

# all contrail components run on master node now


