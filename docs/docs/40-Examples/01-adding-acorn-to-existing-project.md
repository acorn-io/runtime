---
title: Adding Acorn to an Existing Project
---

If you have an existing application / codebase that you want to package as an Acorn App, you will first need to create an `acorn.cue` file.

In a project with an existing Dockerfile it is easy to package as an Acorn App.

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

Create an `acorn.cue` file.

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

Now you can package your app as an Acorn App by building it.
`acorn build .`
