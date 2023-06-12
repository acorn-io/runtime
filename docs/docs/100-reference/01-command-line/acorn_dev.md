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
      --auto-upgrade         Enabled automatic upgrades.
  -b, --bidirectional-sync   In interactive mode download changes in addition to uploading
  -f, --file string          Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                 help for dev
      --help-advanced        Show verbose help text
  -n, --name string          Name of app to create
      --notify-upgrade       If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string        Output API request without creating app (json, yaml)
      --profile strings      Profile to assign default values
      --replace              Replace the app with only defined values, resetting undefined fields to default values
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

