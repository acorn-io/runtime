---
title: Uninstalling
---

## Uninstalling Acorn from Kubernetes

The following command will uninstall the Acorn components along with the Acorn Apps deployed by Acorn. When you run the command you will be shown the components that will be removed and remain.

```shell
acorn uninstall
#Action | Namespace                     | Name                           | Kind                     | API Version                 
#keep   |                               | acorn                          | Namespace                | v1                          
#delete |                               | acorn-system                   | Namespace                | v1                          
#delete |                               | acorn-system                   | ClusterRole              | rbac.authorization.k8s.io/v1
#delete |                               | acorn-system                   | ClusterRoleBinding       | rbac.authorization.k8s.io/v1
#delete |                               | appinstances.internal.acorn.io | CustomResourceDefinition | apiextensions.k8s.io/v1     
#delete |                               | crimson-darkness-ef125bc3-145  | Namespace                | v1                          
#delete |                               | v1.api.acorn.io                | APIService               | apiregistration.k8s.io/v1   
#delete | acorn-system                  | acorn-api                      | Deployment               | apps/v1                     
#delete | acorn-system                  | acorn-api                      | Service                  | v1                          
#delete | acorn-system                  | acorn-config                   | ConfigMap                | v1                          
#delete | acorn-system                  | acorn-controller               | Deployment               | apps/v1                     
#delete | acorn-system                  | acorn-dns                      | Secret                   | v1                          
#delete | acorn-system                  | acorn-system                   | ServiceAccount           | v1                          
#delete | crimson-darkness-ef125bc3-145 | default-pull-ef125bc3-145      | Secret                   | v1                          
#? Do you want to delete/keep the above resources? To delete all resources pass run "acorn uninstall --all" (y/N) 
```

Once confirmed Acorn will remove the listed components.

If you would like to delete all resources you can add the `--all` flag to remove resources that normal uninstall would leave behind.
