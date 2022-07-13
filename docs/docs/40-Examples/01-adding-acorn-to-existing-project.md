---
title: Adding Acorn to an Existing Project
---

If you have an existing application / codebase that you want to package as an Acorn App, you will first need to create an `acorn.cue` file.

In a project with an existing Dockerfile you will need to create an Acorn file that will build the image.

```shell
> ls -l
...
-rw-r--r--  1 user  staff   217B Jun 21 07:54 Dockerfile
-rw-r--r--  1 user  staff    89B May 26 10:05 babel.config.js
drwxr-xr-x  4 user  staff   128B Jun  6 15:04 diagram-source
drwxr-xr-x  9 user  staff   288B Jun  6 15:04 docs
-rw-r--r--  1 user  staff   2.5K Jun 21 07:54 docusaurus.config.js
-rw-r--r--  1 user  staff   1.2K Jun  3 13:19 package.json
-rw-r--r--  1 user  staff   719B May 26 10:05 sidebars.js
drwxr-xr-x  4 user  staff   128B May 31 11:01 src
drwxr-xr-x  5 user  staff   160B Jun  1 09:00 static
-rw-r--r--  1 user  staff   360K May 31 11:01 yarn.lock
...
```

Take a quick look at this very simple Dockerfile here:

```dockerfile
FROM nginx
ADD . .
EXPOSE 80
```

This file describes building a container from the upstream nginx image, it adds the contents of the local directory, and exposes port 80.

We can now translate that to the Acorn file by adding the contents:

```cuelang
containers: {
    docs: {
        build: {
            context: .  // Same as what you would pass to `docker build .`
        }
    }
    expose: "80:80/http"
}
```

This Acorn file describes building the docker container, with the equivalent of `docker build .`. Docker is not needed on the system running Acorn, it will be built on the Kubernetes cluster in the Acorn provided builder.

Now you can package your app as an Acorn App by building it.
`acorn build .`

At the end of this build there will be a long string, you can now deploy the app by running:
`acorn run [IMAGE-SHA]`
