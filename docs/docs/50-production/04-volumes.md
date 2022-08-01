---
title: Volumes
---

## Configuring volumes at runtime

Acorn images can define volumes to see which volumes are available for an Acorn image you can run `--help` on the image.

```shell
acorn run [IMAGE] --help
# ...
#Volumes: my-data
```

The `Volumes` line shows the volumes that will be created as part of the Acorn app deployment. Each volume will use the `default` StorageClass of the cluster and request 10GB of space. This can be customized at runtime by passing `-v` arguments.

|Default| Value|
|-------| -----|
| size | 10G|
|class | default|

### Using dynamically created volumes

To use a dynamically created volume from a StorageClass you can specify the `class`, `size`

```shell
acorn run -v my-data,size=5G,class=fast [IMAGE]
```

With the above command Acorn will create a volume with `5G` using the `fast` storage class in the cluster.

### Using precreated volumes

You can use a precreated volumes by binding the volume at runtime.

```shell
acorn run -v db-data:my-data [IMAGE]
```

This Acorn app will use the volume named `db-data` as it's `my-data` volume.

The volume will match the size and class of the pre-created PV `db-data`.
