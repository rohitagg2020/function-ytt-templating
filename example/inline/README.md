# Cluster Example

In this example, we will install function on the K8s cluster which has crossplane installed.

## Pre-requisite
* K8s cluster with crossplane installed.

```shell
$ kubectl apply -f functions.yaml
$ kubectl apply -f xrd.yaml
$ kubectl apply -f composition.yaml
$kubectl apply -f claim.yaml
```

After applying the claim, if we look into the logs of function pod, we should see the following message:
```"Successfully composed desired resources","source":"Inline","count":3```
