---
title: Update Acorns
---

Acorns can be updated through various methods

- acorn run --update
- acorn run --replace
- acorn update

### run --update
`acorn run --update` will create or update an existing acorn with the provided flags and/or arguments.

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
//start an acorn from a directory
$ acorn run -n awesome-acorn .
awesome-acorn

$ acorn app                   
NAME             IMAGE          HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS                                                            MESSAGE
awesome-acorn    3e23d225e777   1         1            10s ago    http://nginx-awesome-acorn-9ca4278a.local.on-acorn.io => nginx:80    OK

//update the msg arg and add a label
acorn run --update --label label=new -n awesome-acorn -- --msg 2

//navigate to the endpoint to see the updated message
```
:::note the `--` is used to differentiate between flags and acorn args

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
`acorn update` follows the same pattern as `acorn run --update`
