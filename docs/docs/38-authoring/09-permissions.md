---
title: Permissions
---

When writing applications, you can run into a situation where your application needs to interact with on-cluster resources. In a typical Kubernetes environment you would need to go through a somewhat involved process to accomplish this. Luckily, `permissions` is a straight forward Acornfile definition that allows you to simplify that process.

Let's take a look at an example. Here we have a `container`, named API, that we want to grant CRUD operations on the `FooResource` in the application's namespace. For all other namespaces, we want the `container` to only be able to retrieve `FooResources`.

```acorn
containers:{
    api: {
        // ...
        permissions: {
            rules: [{
                verbs: [
                    "get", 
                    "list", 
                    "watch",
                    "create", 
                    "update",
                    "patch", 
                    "delete"
                ]
                apiGroups: [
                    "api.sample.io"
                ]
                resources: [
                    "fooresource"
                ]
            }]
            clusterrules: [{
                verbs: [
                    "get", 
                    "list", 
                    "watch",
                ]
                apiGroups: [
                    "api.sample.io"
                ]
                resources: [
                    "fooresource"
                ]
            }]
        / ...
    }
}
```

:::info
If you're curious, running this Acornfile creates a few `permissions` related things for us, such as a:
- `ServiceAccount` bound to the `container`'s `Deployment`
- `Role` with the `rules` we specified
- `ClusterRole` with the `clusterrules` we specified
- `RoleBinding` with `Role` bound to the `ServiceAccount`
- `ClusterRoleBinding` with the `ClusterRole` bound to the `ServiceAccount`
:::

With this Acornfile, we accomplish our original goal. Breaking down the Acornfile a bit further, we get 5 keywords that are set to define permissions. Let's look at them one at a time.

## Rules
Physically defining the permissions of your application, `rules` get converted into a `Role` that then gets attached to your application's unique `ServiceAccount`. This is only applicable for your application's unique namespace and as a result the permissions will not work in other namespaces.

## ClusterRules
Similar to `rules`, `clusterrules` define the permissions application's namespace but with the added benefit of working in other ones as well. Instead of creating a `Role` that gets attached to your application's `ServiceAccount`, you get a `ClusterRole`. If you would like to allow your application to perform the defined rules in any namespace on the cluster then `clusterrules` are the way to go.

## Verbs
To define what actions your application can perform on a given resource, you define a `verb`. These `verbs` are words that allow you to declaritively define what actions your application can perform on given resources.

:::info
Wondering what verbs are available? Take a look!
- get
- list
- watch
- create
- update
- patch
- delete
- deletecollection
:::

## ApiGroups
When interacting with on-cluster resources, related resources are typically grouped by an `apiGroup`. For the context of Acorn, we need to know what `apiGroup` the resource we're granting permissions for is in. In our original example this was `api.sample.io` and others will typically be in this format.

## Resources
Inside of `apiGroups` you'll find associated `resources`. With this field, you specify which `resources` the `rules` you are creating apply to. In our original example, this was `foo`.