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
acorn dev --name wandering-sound --clone [acorn args]

```

### Options

```
      --args-file string     Default args to apply to run/update command (default ".args.acorn")
      --auto-upgrade         Enabled automatic upgrades.
  -b, --bidirectional-sync   In interactive mode download changes in addition to uploading
      --clone                Clone the vcs repository and infer the build context for the given app allowing for local development
  -f, --file string          Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                 help for dev
      --help-advanced        Show verbose help text
  -n, --name string          Name of app to create
      --notify-upgrade       If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string        Output API request without creating app (json, yaml)
      --replace              Replace the app with only defined values, resetting undefined fields to default values
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

