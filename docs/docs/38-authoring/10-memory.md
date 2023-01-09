---
title: Memory
---
Setting the memory via the Acornfile uses the `memory` property is settable for all `workloads` (`containers` and `jobs`). Check out the [memory reference documentation](../100-reference/06-memory.md) for more information on ways to set memory.

```acorn
containers: {
    nginx: {
        image: "nginx"
        ports: publish: "80/http"
        files: {
            "/usr/share/nginx/html/index.html": "<h1>My first Acorn!</h1>"
        }
        memory: 512Mi
    }
}
```

:::tip
The `memory` property can be abbreviated to `mem` in the file.
:::
