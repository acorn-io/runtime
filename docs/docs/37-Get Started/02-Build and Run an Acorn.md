---
title: Build and Running Acorn Apps
---

## Adding Acorn to your containerized app

Containerized apps typically have a Dockerfile and are packaged as a container by running `docker build .`  as a part of the CI pipeline.

A simple Dockerfile looks like:

```dockerfile
FROM nginx
ADD . .
EXPOSE 80
```

You will get the most benefit in your workflows if you use Acorn to build the image.

To do that, create a minimal `Acornfile` in the same directory as the Dockerfile:

```cue
containers: {
  app: {
    build: {
      context: "."
    }
  }
}
```

This Acornfile defines a container named 'app' and builds it. It is the equivalent of running `docker build .`.

Run the command below to build the Acornfile:

`acorn build .`

This will build an image and make it available to run on the cluster.

![Build output](/img/build_output.png)

## Running the acorn

Now that the image is built, you can now run that image:

`acorn run 53f8ba8d473c92d093ae17958ab24265775c2d8d8559554fc04d743bd7f7d589`

Where the `53f8ba8d473c92d093ae17958ab24265775c2d8d8559554fc04d743bd7f7d589` comes from the last line of your `acorn build .` command.

The output of the run command will be the app name which will look like: `snowy-dawn`.

## Checking the status of running Acorn apps

### Viewing apps

Once the app has been deployed by the run command, you can check the status of the application by running:

`acorn apps`

This will show the status of all running apps, along with the endpoints and any errors.

### Viewing containers

To check on the status of individual containers within the Acorn app, you can run:

`acorn containers` or specific to the app `acorn containers [APP-NAME]`

This will show you the status of the individual containers.

### Viewing all resources

In order to view all of the resources defined in your acorn namepace, you can use:

`acorn all`

If you would like to see apps that are stopped you can use:

`acorn all -a`

To watch what is happening in the environment you can use the OS `watch` command.

`watch acorn all`

### Viewing logs of all containers in the Acorn app

To view the logs of your running application you can run:
`acorn logs [APP-NAME]`

If you would like the logs to continue streaming, you can add the `-f` to follow the logs.
