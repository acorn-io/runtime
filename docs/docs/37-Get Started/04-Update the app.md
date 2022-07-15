---
title: Updating an Acorn App Image
---

## Updating a running Acorn

Making changes to a running Acorn are all additive, and do not require all arguments be passed on each update.

### Build a new image

Once you make changes to the source code of your application or changes to the `Acornfile` you will need to rebuild the image.

`aconr build .`

There will be a new SHA available.

### Upgrading the image

If a new Acorn image is available for the application, it can be updated by running:

`acorn update -i [NEW-IMAGE] [APP-NAME]`

The new image can be the sha or tag of an updated image.

If you will be doing a lot of build/updates you can use:

```shell
> image=$(acorn build .) && acorn update -i ${image} [APP-NAME]
```
