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

## Using pre-existing volumes

You can use a pre-existing volumes by binding the volume at runtime.
The volume can be referenced either by its PersistentVolume name in Kubernetes, or by its name in Acorn (displayed in the output of `acorn volume`).
In this example, the new Acorn app uses an old volume called `data` that an app called `db` used. It uses it as its `my-data` volume.

```
$ acorn volume
NAME      APP-NAME   BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED
db.data   db.data    data           1G         local-path     bound     RWO            23s ago

$ acorn run -v "db.data:my-data" -n my-new-app [IMAGE]
```

The volume will match the size and class of the pre-existing volume `db.data`.
Once the old volume is consumed by the new app, it will be renamed.

```
$ acorn volume
NAME                 APP-NAME             BOUND-VOLUME   CAPACITY   VOLUME-CLASS   STATUS    ACCESS-MODES   CREATED
my-new-app.my-data   my-new-app.my-data   my-data-bind   1G         local-path     bound     RWO            2m16s ago
```

A pre-existing volume can only be bound to a new app if the new app is created in the same Acorn project as the old app that previously used the volume.

At this time, volumes created outside of Acorn cannot be bound to an Acorn app.
