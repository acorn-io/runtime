---
title: Update Acorns
---

There are multiple ways to update an Acorn, including using the following methods:

- `acorn run --update`
- `acorn run --replace`
- `acorn update`

### run --update
By running `acorn run --update`, you can create a new Acorn or update an existing one with the specified flags and/or arguments.

Take this Acornfile
```Acornfile
containers: {
  nginx: {
    image: "nginx"
    ports: publish: "80/http"
    files: {
      "/usr/share/nginx/html/index.html": "<h1>My " + args.msg + " Acornfile</h1>"
    }
    mem: 128Mi
    labels: {
                key: "test-label"
            }
  }
}
args: {
    // new arg
    msg: "1"
}
```

```bash
#start an acorn from a directory
$ acorn run -n awesome-acorn .
awesome-acorn

$ acorn app                   
NAME             IMAGE          HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS                                                            MESSAGE
awesome-acorn    3e23d225e777   1         1            10s ago    http://nginx-awesome-acorn-9ca4278a.local.oss-acorn.io => nginx:80    OK

#update the msg arg and add a label
acorn run --update --label label=new -n awesome-acorn -- --msg 2

#navigate to the endpoint to see the updated message
```

:::note
The purpose of using -- is to distinguish between command-line options (flags) and arguments for the Acorn.
:::


### acorn --replace
Similarly to `acorn run --update`, `acorn run --replace` will create or update an existing acorn with ONLY the provided flags and args. Any previous modifications will be replaced.

If we attempt a replace on the previous acorn, we should see the label be dropped
```bash
$ acorn run --replace -n awesome-acorn -- --msg 3

$ acorn app awesome-acorn -o yaml                  
---
metadata:
  creationTimestamp: "2023-03-23T20:11:11Z"
  generation: 3
  name: awesome-acorn
  namespace: acorn
  ...
status:
  appImage:
    acornfile: |
      containers: {
        nginx: {
          image: "nginx"
          ports: publish: "80/http"
          files: {
            "/usr/share/nginx/html/index.html": "<h1>My " + args.msg + " Acornfile</h1>"
          }

          mem: 128Mi
          labels: {
                      key: "test-value"
                  }
        }
      }
   ...
```

### acorn update
Both `acorn update` and `acorn run --update` have identical functionality, with the former being an alias for the latter.


### More Examples

```shell
# update a currently running acorn's image and modify its' acorn args
acorn run --update -n my-acorn --image my-new-image -- --acorn-arg newArg

#update a currently running acorn from the current dir
acorn update -n my-acorn .
```
