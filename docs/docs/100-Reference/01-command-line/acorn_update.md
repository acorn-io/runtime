---
title: "acorn update"
---
## acorn update

Update a deployed app

```
acorn update [flags] APP_NAME [deploy flags]
```

### Options

```
      --dangerous         Automatically approve all privileges requested by the application
      --expose strings    In cluster expose ports of an application (format [public:]private) (ex 81:80)
  -f, --file string       Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help              help for update
      --image string      
  -l, --link strings      Link external app as a service in the current app (format app-name:service-name)
  -n, --name string       Name of app to create
      --profile strings   Profile to assign default values
  -p, --publish strings   Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all       Publish all (true) or none (false) of the defined ports of application
  -s, --secret strings    Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -v, --volume strings    Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 

