---
title: "acorn run"
---
## acorn run

Run an app from an image or Acornfile

```
acorn run [flags] IMAGE|DIRECTORY [acorn args]
```

### Examples

```

 # Build and run from a directory
   acorn run .

 # Run from an image
   acorn run ghcr.io/acorn-io/hello-world

 # Automatic upgrades
   # Automatic upgrade for an app will be enabled if '#', '*', or '**' appears in the image's tag. Tags will be sorted according to the rules for these special characters described below. The newest tag will be selected for upgrade.
   
   # '#' denotes a segment of the image tag that should be sorted numerically when finding the newest tag.

   # This example deploys the hello-world app with auto-upgrade enabled and matching all major, minor, and patch versions:
   acorn run myorg/hello-world:v#.#.#

   # '*' denotes a segment of the image tag that should sorted alphabetically when finding the latest tag.
  
   # In this example, if you had a tag named alpha and a tag named zeta, zeta would be recognized as the newest:
   acorn run myorg/hello-world:*

   # '**' denotes a wildcard. This segment of the image tag won't be considered when sorting. This is useful if your tags have a segment that is unpredictable.
   
   # This example would sort numerically according to major and minor version (i.e. v1.2) and ignore anything following the "-":
   acorn run myorg/hello-world:v#.#-**

   # NOTE: Depending on your shell, you may see errors when using '*' and '**'. Using quotes will tell the shell to ignore them so acorn can parse them:
   acorn run "myorg/hello-world:v#.#-**"

   # Automatic upgrades can be configured explicitly via a flag.

   # In this example, the tag will always be "latest", but acorn will periodically check to see if new content has been pushed to that tag:
   acorn run --auto-upgrade myorg/hello-world:latest

   # To have acorn notify you that an app has an upgrade available and require confirmation before proceeding, set the notify-upgrade flag:
   acorn run --notify-upgrade myorg/hello-world:v#.#.# myapp

   # To proceed with an upgrade you've been notified of:
   acorn update --confirm-upgrade myapp
```

### Options

```
      --auto-upgrade         Enabled automatic upgrades.
  -b, --bidirectional-sync   In interactive mode download changes in addition to uploading
  -i, --dev                  Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit
  -f, --file string          Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                 help for run
      --help-advanced        Show verbose help text
  -n, --name string          Name of app to create
      --notify-upgrade       If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string        Output API request without creating app (json, yaml)
      --profile strings      Profile to assign default values
  -q, --quiet                Do not print status
      --wait                 Wait for app to become ready before command exiting (default: true)
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

