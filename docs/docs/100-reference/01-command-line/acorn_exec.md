---
title: "acorn exec"
---
## acorn exec

Run a command in a container

### Synopsis

Run a command in a container

```
acorn exec [flags] APP_NAME|CONTAINER_NAME CMD
```

### Options

```
  -c, --container string     Name of container to exec into
  -d, --debug-image string   Use image as container root for command
  -h, --help                 help for exec
  -i, --interactive          Not used
  -t, --tty                  Not used
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

