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
      --annotation strings        Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --auto-upgrade              Enabled automatic upgrades.
      --confirm-upgrade           When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade.
  -e, --env strings               Environment variables to set on running containers
      --expose strings            In cluster expose ports of an application (format [public:]private) (ex 81:80)
  -f, --file string               Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                      help for update
      --image string              
      --interval string           If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)
  -l, --label strings             Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --link strings              Link external app as a service in the current app (format app-name:container-name)
  -n, --name string               Name of app to create
      --notify-upgrade            If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string             Output API request without creating app (json, yaml)
      --profile strings           Profile to assign default values
  -p, --publish strings           Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all               Publish all (true) or none (false) of the defined ports of application
      --pull                      Re-pull the app's image, which will cause the app to re-deploy if the image has changed
      --replace                   Toggle replacing update, resetting undefined fields to default values
  -s, --secret strings            Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
      --target-namespace string   The name of the namespace to be created and deleted for the application resources
  -v, --volume stringArray        Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)
```

### Options inherited from parent commands

```
  -A, --all-namespaces      Namespace to work in
      --context string      Context to use in the kubeconfig file
      --debug               Enable debug logging
      --debug-level int     Debug log level (valid 0-9) (default 7)
      --kubeconfig string   Location of a kubeconfig file
      --namespace string    Namespace to work in (default "acorn")
```

### SEE ALSO

* [acorn](acorn.md)	 - 

