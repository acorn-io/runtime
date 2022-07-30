---
title: Updating an Acorn App Image
---

Making changes to a running Acorn is all additive, and does not require all arguments to be passed on each update.

### Building a new image

Once you make changes to the source code of your application or changes to the `Acornfile` you will need to rebuild the image.

`acorn build .`

There will be a new SHA available.

### Upgrading the image

If a new Acorn image is available for the application, it can be updated by running:

`acorn update --image [NEW-IMAGE] [APP-NAME]`

The new image can be the SHA or tag of an updated image.

If you will be doing a lot of builds/updates you can use:

```shell
image=$(acorn build .) && acorn update -i ${image} [APP-NAME]
```
