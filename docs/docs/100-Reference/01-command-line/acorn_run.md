---
title: "acorn run"
---
## acorn run

Run an app from an image or Acornfile

```
acorn run [flags] IMAGE|DIRECTORY [acorn args]
```

### Options

```
  -b, --bidirectional-sync   In interactive mode download changes in addition to uploading
      --dangerous            Automatically approve all privileges requested by the application
  -d, --dns strings          Assign a friendly domain to a published container (format public:private) (ex: example.com:web)
  -f, --file string          Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                 help for run
  -i, --interactive          Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit
  -l, --link strings         Link external app as a service in the current app (format app-name:service-name)
  -n, --name string          Name of app to create
      --profile strings      Profile to assign default values
  -p, --publish strings      Publish exposed port of application (format [public:]private) (ex 81:80)
  -P, --publish-all          Publish all exposed ports of application
  -s, --secret strings       Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -v, --volume strings       Bind an existing volume (format existing:vol-name) (ex: pvc-name:app-data)
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

