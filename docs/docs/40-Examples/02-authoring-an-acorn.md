---
title: Authoring an Acorn
---

## Structure of a new standalone Acorn app

1. Create a new top level directory with the apps name.
1. There needs to be an `acorn.cue` file and a `README.md` for each acorn.

    a.) acorn.cue will specify the definition for the Acorn.
    b.) README.md will document how to use the acorn. It should include all relevant details on how to use the Acorn in a production setting.

```shell
> mkdir <APP>
> touch ./<APP>/acorn.cue ./<APP>/README.md
> ls -l <APP>
ll APP
total 0
-rw-r--r--  1 owner group 0B Jun 21 10:53 README.md
-rw-r--r--  1 owner group 0B Jun 21 10:53 acorn.cue
```

## Acorn Development Workflow

### Iterating quickly with Dev mode

When developing a new Acorn it is best to have a shell and an editor open with the `acorn.cue` file.

Starting with a simple `acorn.cue` file like:

```cue
containers: "my-app": image: "nginx"
```

in the shell start dev mode:

```shell
> cd ./APP/
> acorn dev .
```

This will build the acorn and start it in a live reload mode along with streaming all container logs to the screen. Each change to the `acorn.cue` file will cause Acorn to automatically rebuild. So adding a port for instance:

```cue
containers: {
    "my-app": {
        image: "nginx"
        ports: "80:80"
    }
}
```

Will cause Acorn to reload with the new port definition.

### Quickly render template outputs

When developing configuration files and you want to see if you are getting the right output without having to wait for containers to launch, you can use `acorn dev render .`

Render mode also lets you pass deployment arguments so that you can verify the files are being rendered correctly.

The output is formated in JSON so you can pipe to `jq` to narrow down the view.

```shell
> cd ./redis
> acorn dev render . | jq -r '.containers."redis-0-0".files."/acorn/redis-ping-local-liveness.sh"'
#!/bin/sh
res=$(timeout -s 3 ${1} /usr/local/bin/redis-cli -h localhost -p 6379 ping)
if ["$?" -eq "124"]; then
  echo "Timed out"
  exit 1
fi
if ["$response" != "PONG"]; then
  echo "${response}"
  exit 1
fi
```

## Best Practices

1. Use Docker library images if available.
1. Use specific version tags of images, at least to the minor in semver.
1. Use automatically generated secrets, and allow the user to specify values if needed.
1. Prefer to handle the application in cuelang vs. string interpolations.
