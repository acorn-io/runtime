---
title: "acorn dev"
---
## acorn dev

Build and run an app in development mode

### Synopsis

Build and run an app in development mode

```
acorn dev [flags] DIRECTORY
```

### Options

```
  -d, --dns strings      Assign a friendly domain to a published container (format public:private) (ex: example.com:web)
  -f, --file string      Name of the dev file (default "DIRECTORY/acorn.cue")
  -h, --help             help for dev
  -l, --link strings     Link external app as a service in the current app (format app-name:service-name)
  -n, --name string      Name of app to create
  -s, --secret strings   Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -v, --volume strings   Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)
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
* [acorn dev render](acorn_dev_render.md)	 - Evaluate and display an acorn.cue with deploy params

