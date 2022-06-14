---
title: Uninstalling
---

Uninstalling acorn is straightforward:


First, delete the Acorn ApiService, then delete the acorn-system namespace:
```
kubectl delete apiservices.apiregistration.k8s.io v1.api.acorn.io
kubectl delete namespace acorn-system
```

These two commands will delete the acorn control plane, but leave the resources related to running acorn applications intact.

The following will remain: 
- The `appinstance.acorn.io` CRD
- The `acorn` namespace and any resources created in it. This is where `appinstance` CRs reside.
- The workloads, services, and other resources created for running your acorn apps. These are contained in namespaces created by acorn with names like `bold-water-624fb33e`.

You can delete the above to fully purge your cluster of all acorn related resources. To delete all namespaces created by acorn, you can run:
```
kubectl delete namespace -l acorn.io/managed=true
```

