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

Running an app
 - Build and run from a directory
	acorn run --name my-app .

 - Run from an image
	acorn run --name my-app ghcr.io/acorn-io/hello-world

Updating an app
 - Rebuild an app from the current directory (see 'acorn dev' to do this dynamically)
	acorn run --update --name my-app .

 - Change an app's image
	acorn run --update --image ghcr.io/acorn-io/hello-world:v1.0.2 --name my-app

Automatic upgrades
 - Automatic upgrade for an app will be enabled if '#', '*', or '**' appears in the image's tag. Tags will be sorted according to the rules for these special characters described below. The newest tag will be selected for upgrade.
	'#' denotes a segment of the image tag that should be sorted numerically when finding the newest tag.
	'*' denotes a segment of the image tag that should sorted alphabetically when finding the latest tag.
	'**' denotes a wildcard. This segment of the image tag won't be considered when sorting. This is useful if your tags have a segment that is unpredictable.

 - Deploy the hello-world app with auto-upgrade enabled and matching all major, minor, and patch versions:
	acorn run ghcr.io/acorn-io/hello-world:v#.#.#

Publish Port Syntax
 - Publish port 80 for any containers that define it as a port
	acorn run -p 80 .

 - Publish container "myapp" using the hostname app.example.com
	acorn run --publish app.example.com:myapp .

Link Syntax
 - Link the running acorn application named "mydatabase" into the current app, replacing the container named "db"
	acorn run --link mydatabase:db .

Secret Syntax
- Bind the acorn secret named "mycredentials" into the current app, replacing the secret named "creds"
	acorn run --secret mycredentials:creds .

Volume Syntax
 - Create the volume named "mydata" with a size of 5 gigabyes and using the "fast" storage class
	acorn run --volume mydata,size=5G,class=fast .
 
- Bind the acorn volume named "mydata" into the current app, replacing the volume named "data"
	acorn run --volume mydata:data .
```

### Options

```
      --annotation strings      Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --args-file string        Default args to apply to run/update command (default ".args.acorn")
      --auto-upgrade            Enabled automatic upgrades.
  -b, --bidirectional-sync      In interactive mode download changes in addition to uploading
      --compute-class strings   Set computeclass for a workload in the format of workload=computeclass. Specify a single computeclass to set all workloads. (ex foo=example-class or example-class)
      --dangerous               Automatically approve all privileges requested by the application
  -i, --dev                     Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit
  -e, --env strings             Environment variables to set on running containers
      --env-file string         Default env vars to apply (default ".acorn.env")
  -f, --file string             Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                    help for run
      --interval string         If configured for auto-upgrade, this is the time interval at which to check for new releases (ex: 1h, 5m)
  -l, --label strings           Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --link strings            Link external app as a service in the current app (format app-name:container-name)
  -m, --memory strings          Set memory for a workload in the format of workload=memory. Only specify an amount to set all workloads. (ex foo=512Mi or 512Mi)
  -n, --name string             Name of app to create
      --notify-upgrade          If true and the app is configured for auto-upgrades, you will be notified in the CLI when an upgrade is available and must confirm it
  -o, --output string           Output API request without creating app (json, yaml)
  -p, --publish strings         Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all             Publish all (true) or none (false) of the defined ports of application
  -q, --quiet                   Do not print status
      --region string           Region in which to deploy the app, immutable
      --replace                 Replace the app with only defined values, resetting undefined fields to default values
  -s, --secret strings          Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -u, --update                  Update the app if it already exists
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

