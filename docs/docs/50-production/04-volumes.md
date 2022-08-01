---
title: Volumes
---

## Configuring volumes at runtime

Acorn Images can define volumes to see which volumes are available for an Acorn Image you can run `--help` on the image.

```shell
acorn run [IMAGE] --help
# ...
#Volumes: my-data
```

The `Volumes` line shows the volumes that will be created as part of the Acorn App deployment. Each volume will use the `default` StorageClass of the cluster and request 10GB of space. This can be customized at runtime by passing `-v` arguments.

|Default| Value|
|-------| -----|
| size | 10G|
|class | default|

### Using dynamically created volumes

To use a dynamically created volume from a StorageClass you can specify the `class`, `size`

```shell
acorn run -v my-data [IMAGE]
```

### Using precreated volumes

You can use a precreated volumes by binding the volume at runtime.

```shell
acorn run -v db-data:my-data [IMAGE]
```

This Acorn App will use the volume named `db-data` as it's `my-data` volume.
