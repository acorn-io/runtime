---
title: Volumes
---

## Configuring volumes at runtime

Acorn images can define volumes. To see which volumes are available for an Acorn image you can run `--help` on the image.

```shell
acorn run [IMAGE] --help
# ...
Volumes: my-data
```

The `Volumes` line shows the volumes that will be created as part of the Acorn app deployment. Unless otherwise [specified in the Acorn image](38-authoring/04-volumes.md) and if no default volume class is specified, each volume will be created using the following default values:

|Field       |Default Value|
|----------- | -----|
| size       | 10G |
| class      | default |
| accessMode | readWriteOnce |

These values can be customized at runtime by passing `-v` arguments. For example, the following command will cause the "my-data" volume to be created with `5G` of storage and using the `fast` volume class:

```shell
acorn run -v my-data,size=5G,class=fast [IMAGE]
```

The volume class used, including the default, may have restrictions on the size of volumes created or the access modes available. If your volume uses a class that is not available or uses class settings that violate its rules, then will not run. A descriptive error will be produced to explain any failures.

You can see a list of available volume classes and their restrictions, if any, with the [`acorn offerings volumeclasses`](100-reference/01-command-line/acorn_offerings_volumeclasses.md) command.

## Using precreated volumes

You can use a precreated volumes by binding the volume at runtime.

```shell
acorn run -v db-data:my-data [IMAGE]
```

This Acorn app will use the volume named `db-data` as its `my-data` volume.

The volume will match the size and class of the pre-created PV `db-data`.
