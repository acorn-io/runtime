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
      --auto-upgrade      Enabled automatic upgrades.
      --confirm-upgrade   When an auto-upgrade app is marked as having an upgrade available, pass this flag to confirm the upgrade. Used in conjunction with --notify-upgrade.
  -f, --file string       Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help              help for update
      --help-advanced     Show verbose help text
      --image string      Acorn image name
      --notify-upgrade    If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string     Output API request without creating app (json, yaml)
      --profile strings   Profile to assign default values
      --pull              Re-pull the app's image, which will cause the app to re-deploy if the image has changed
  -q, --quiet             Do not print status
      --wait              Wait for app to become ready before command exiting (default: true)
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

