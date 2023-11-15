---
title: "acorn update"
---
## acorn update

Update a deployed Acorn

```
acorn update [flags] ACORN_NAME [deploy flags]
```

### Examples

```

  # Change the image on an Acorn called "my-app"
    acorn update --image <new image> my-app

  # Change the image on an Acorn called "my-app" to the contents of the current directory (which must include an Acornfile)
    acorn update --image . my-app

  # Enable auto-upgrade on an Acorn called "my-app"
    acorn update --auto-upgrade my-app
```

### Options

```
      --annotation strings      Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --args-file string        Default args to apply to run/update command (default ".args.acorn")
      --auto-upgrade            Enabled automatic upgrades.
      --compute-class strings   Set computeclass for a workload in the format of workload=computeclass. Specify a single computeclass to set all workloads. (ex foo=example-class or example-class)
      --confirm-upgrade         When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade.
      --dangerous               Automatically approve all privileges requested by the application
  -e, --env strings             Environment variables to set on running containers
      --env-file string         Default env vars to apply to update command
  -f, --file string             Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                    help for update
      --image string            Acorn image name
      --interval string         If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)
  -l, --label strings           Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --link strings            Link external app as a service in the current app (format app-name:container-name)
  -m, --memory strings          Set memory for a workload in the format of workload=memory. Only specify an amount to set all workloads. (ex foo=512Mi or 512Mi)
      --notify-upgrade          If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string           Output API request without creating app (json, yaml)
  -p, --publish strings         Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all             Publish all (true) or none (false) of the defined ports of application
      --pull                    Re-pull the app's image, which will cause the app to re-deploy if the image has changed
  -q, --quiet                   Do not print status
      --region string           Region in which to deploy the app, immutable
  -s, --secret strings          Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -v, --volume stringArray      Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)
      --wait                    Wait for app to become ready before command exiting (default: true)
```

### Options inherited from parent commands

```
      --config-file string   Path of the acorn config file to use
      --debug                Enable debug logging
      --debug-level int      Debug log level (valid 0-9) (default 7)
      --kubeconfig string    Explicitly use kubeconfig file, overriding the default context
  -j, --project string       Project to work in
```

### SEE ALSO

* [acorn](acorn.md)	 - 

