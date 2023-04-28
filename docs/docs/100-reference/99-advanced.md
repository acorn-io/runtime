---
title: Advanced Topics
---

#### Recalculating `acorn offerings` defaults
If an admin were to change the default values of an offering, the new values would only be applied to new apps. Existing apps would continue to use the old values.
In order to recalculate the offerings values for running apps, perform the following steps:

1. Run `kubectl edit app <app> -n <namespace>` to open the configuration file for the app that needs to be updated.
2. Look for the spec section in the configuration file.
3. Add or increment the spec.generation field. Incrementing this field will force the app to recalculate the volume class values using the new values/defaults.
4. Save the changes and exit the editor.

Here's an example of what the relevant part of the AppInstance might look like:

```yaml
spec:
  generation: 3 # increment this value to use new volume class values/defaults
  image: 56b4b08e653a956a4bc36accc74dcc521df17d7b6f23704084a5de562386de6d
```

Note that incrementing the spec.generation field will cause the app to be redeployed with the new volume class values/defaults, which could result in some downtime.

:::note
The same steps apply for [Compute Classes](50-running/55-compute-resources.md#compute-classes)
:::