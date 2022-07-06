---
title: Running Acorns
---

## Running existing Acorn

If you would like to run an acorn from a registry the command is:

`acorn run registry.example.com/myorg/iYou could unpublish the mage`

There is no need to pull the image ahead of time if the image is not on the host. It will be pulled when acorn goes to run the image.

To see what arguments are available to customize the Acorn add `--help` after the image name.

`acorn run registry.example.com/myorg/image --help`

To pass values:

`acorn run registry.example.com/myorg/image --a-false-bool=false --replicas 2`

### Run in interactive mode

Sometimes it is useful to run an Acorn in interactive mode. Running in this mode streams the logs/status to the foreground and stops the app on exit.

`acorn run -i registry.example.com/myorg/image`

## From Dockerfile

If the project already containerized and has an Acorn file you can build and run your image locally. If you need to add an Acorn file see the docs here...

First build the Acorn

`acorn build .`

This will build an image and make it available to run on the cluster. The resulting image will be known by it's 'sha' address. If you want to capture that output you can do:

```shell
> image=$(acorn build .)
> acorn run ${image}
```

If you want to publish your image you can build and tag your image in a single command with the following:

`acorn build -t registry.example.com/myorg/image:version .`

## Runtime configurations

By design authors will build Acorns with sane defaults for common usecases, but every deployment has some organization/environment specific configurations. Acorn allows consumers to customize behavior at runtime.

### Publishing ports

Network ports are defined per container within the Acorn file. At runtime the user can decide to publish those ports outside of the cluster to the rest of the environment or world.

#### All Ports

If you would like to expose all the ports of an Acorn run:

`acorn run -P registry.example.com/myorg/image`

The `-P` option publishes all ports that the Acorn exposes.

#### Selectively publish TCP ports

To learn which ports are available to publish, look at the image help.

`acorn run registry.example.com/myorg/image --help`

There will be a line `Ports: ...` that outlines the ports. To expose the port:

`acorn run -p 3306:3306/tcp registry.example.com/myorg/image`

#### Publish HTTP Ports

To publish an HTTP port, you use the `-d` option on the run subcommand.

`acorn run -d my-app.example.com:frontend:frontend registry.example.com/myorg/image`

### Binding secrets

To securely manage sensitive information while running Acorns the best practice is to use secrets. To accomplish this, the user needs to pre-create secrets before running the app.

#### Show which secrets exist in the Acorn

To see which secrets will be created when the Acorn is deployed pass the `--help` flag on the Acorn image.

`acorn run registry.example.com/myorg/image --help`

There will be a `Secrets` line that lists the names of the secrets in the Acorn.

#### Binding a secret at runtime

When running the Acorn you can bind in a secret with the `-s` option.

`acorn run -s my-predefined-creds:user-creds registry.example.com/myorg/image`

When this Acorn runs it will use the values in the `my-predefined-creds` secret.

### Passing complex arguments

The easiest way to pass complex arguments to create a file in the local directory and pass with the `@` syntax.

config.yaml

```yaml
my:
  map:
    config: value
```

`acorn run registry.example.com/myorg/image --config @config.yaml`

## Interacting with running app

### Viewing all resources

In order to view all of the resources defined in your acorn namepace, you can use:

`acorn all`

If you would like to see apps that are stopped you can use:

`acorn all -a`

To watch what is happening in the environment you can use the OS `watch` command.

`watch acorn all`

### Viewing logs

To view the logs of your running application you can run:
`acorn logs [APP-NAME]`

If you would like the logs to continue streaming, you can add the `-f` to follow the logs.

### Executing commands in side a container

To execute commands in a running Acorn container, you can do:

`acorn exec [APP-NAME]`

You will be prompted for which container if there is more then one running.

If you know the container name you can specify it with the `-c` option.

`acorn exec -c web-01 [APP-NAME]`

## Linking with other Acorns

Acorns can be linked with other running acorns at run time to provide supporting services. For instance if you have an Acorn running Postgresql, it can be used to provide the db service another app.

If you have an Acorn that defines a `web` container and a `redis` container, you can consume a separate Acorn to provide the redis service from an already running Acorn.

`acorn run -l my-other-redis-acorn:redis [IMAGE]`

In the above example the container service from the running Acorn will be available within the new Acorn as `redis`. Your new instance will be able to resolve the `redis` name and it will connect to the remote service defined by the link.

## Updating a running Acorn

Making changes to a running Acorn are all additive, and do not require all arguments be passed on each update.

### Upgrading the image

If a new Acorn image is available for the application, it can be updated by running:

`acorn update -i [NEW-IMAGE] [APP-NAME]`

The new image can be the sha or tag of an updated image.

### Updating parameters

Deployed Acorns can have their parameters changed through the update command. Depending on the parameters being updated it is possible network connectivity maybe lost or containers restarted.

When updating args for the Acorn app, the behavior will be dependent on how the Acorn app was designed/written. Look for documentation from the Acorn app author to understand what is possible and how to operate the Acorn app.

If an Acorn was deployed like:

```shell
> acorn run -d my-app.test.example.com:web [IMAGE] --replicas 3 --cluster-mode active-active
purple-field
```

The app DNS can be updated by running:
`acorn update -d my-app.example.com:web purple-field`

Only the argument being changed needs to be passed in.

## Deleting an Acorn app

Deleting an Acorn app can be accomplished by running:

`acorn rm [APP-NAME]`

This will stop and delete all jobs, containers, and networking services. Volumes and secrets will not be removed by deleting the Acorn app. Those must be cleaned up using the `acorn rm` command for those items.
