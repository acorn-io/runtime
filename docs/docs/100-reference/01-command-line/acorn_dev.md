---
title: "acorn dev"
---
## acorn dev

Run an app from an image or Acornfile in dev mode or attach a dev session to a currently running app

```
acorn dev [flags] IMAGE|DIRECTORY [acorn args]
```

### Examples

```

acorn dev <IMAGE>
acorn dev .
acorn dev --name wandering-sound
acorn dev --name wandering-sound <IMAGE>

```

### Options

```
      --annotation strings        Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --auto-upgrade              Enabled automatic upgrades.
  -b, --bidirectional-sync        In interactive mode download changes in addition to uploading
      --compute-class strings     Set computeclass for a workload in the format of workload=computeclass. Specify a single computeclass to set all workloads. (ex foo=example-class or example-class)
  -e, --env strings               Environment variables to set on running containers
  -f, --file string               Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                      help for dev
      --interval string           If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)
  -l, --label strings             Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --link strings              Link external app as a service in the current app (format app-name:container-name)
  -m, --memory strings            Set memory for a workload in the format of workload=memory. Only specify an amount to set all workloads. (ex foo=512Mi or 512Mi)
  -n, --name string               Name of app to create
      --notify-upgrade            If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string             Output API request without creating app (json, yaml)
      --profile strings           Profile to assign default values
  -p, --publish strings           Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all               Publish all (true) or none (false) of the defined ports of application
  -q, --quiet                     Do not print status
  -s, --secret strings            Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
      --target-namespace string   The name of the namespace to be created and deleted for the application resources
  -v, --volume stringArray        Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)
```

### Options inherited from parent commands

```
  -A, --all-projects        Use all known projects
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Explicitly use kubeconfig file, overriding current project
  -j, --project string      Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

