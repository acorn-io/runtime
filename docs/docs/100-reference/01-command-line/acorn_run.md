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
# Publish and Expose Port Syntax
  # Publish port 80 for any containers that define it as a port
  acorn run -p 80 .

  # Publish container "myapp" using the hostname app.example.com
  acorn run --publish app.example.com:myapp .

  # Expose port 80 to the rest of the cluster as port 8080
  acorn run --expose 8080:80/http .

# Labels and Annotations Syntax
  # Add a label to all resources created by the app
  acorn run --label key=value .

  # Add a label to resources created for all containers
  acorn run --label containers:key=value .

  # Add a label to the resources created for the volume named "myvolume"
  acorn run --label volumes:myvolume:key=value .

# Link Syntax
  # Link the running acorn application named "mydatabase" into the current app, replacing the container named "db"
  acorn run --link mydatabase:db .

# Secret Syntax
  # Bind the acorn secret named "mycredentials" into the current app, replacing the secret named "creds". See "acorn secrets --help" for more info
  acorn run --secret mycredentials:creds .

# Volume Syntax
  # Create the volume named "mydata" with a size of 5 gigabyes and using the "fast" storage class
  acorn run --volume mydata,size=5G,class=fast .

  # Bind the acorn volume named "mydata" into the current app, replacing the volume named "data", See "acorn volumes --help for more info"
  acorn run --volume mydata:data .
```

### Options

```
      --annotation strings   Add annotations to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
  -b, --bidirectional-sync   In interactive mode download changes in addition to uploading
  -i, --dev                  Enable interactive dev mode: build image, stream logs/status in the foreground and stop on exit
  -e, --env strings          Environment variables to set on running containers
      --expose strings       In cluster expose ports of an application (format [public:]private) (ex 81:80)
  -f, --file string          Name of the build file (default "DIRECTORY/Acornfile")
  -h, --help                 help for run
  -l, --label strings        Add labels to the app and the resources it creates (format [type:][name:]key=value) (ex k=v, containers:k=v)
      --link strings         Link external app as a service in the current app (format app-name:container-name)
  -n, --name string          Name of app to create
  -o, --output string        Output API request without creating app (json, yaml)
      --profile strings      Profile to assign default values
  -p, --publish strings      Publish port of application (format [public:]private) (ex 81:80)
  -P, --publish-all          Publish all (true) or none (false) of the defined ports of application
  -s, --secret strings       Bind an existing secret (format existing:sec-name) (ex: sec-name:app-secret)
  -v, --volume stringArray   Bind an existing volume (format existing:vol-name,field=value) (ex: pvc-name:app-data)
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

