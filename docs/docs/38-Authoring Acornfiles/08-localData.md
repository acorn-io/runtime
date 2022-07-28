---
title: localData
---

## localData

The `localData` top level key is used by the Acorn author to store default values for the application. The entire struct is freeform below the top level and it's up to the author to decide how it needs to be structured. Keys in this block should all be camelCased.

```cue
containers:{
    frontend: {
        // ...
        env: {
            "MY_IMPORTANT_SETTING": localData.myApp.frontendConfig.key
        }
        // ...
    }
    database: {
        // ...
        env: {
            "MY_DATABASE_NAME": localData.myApp.databaseConfig.name
        }
        // ...
    }
}
localData: {
    myApp:{
        frontendConfig: {
            key: "value"
        }
        databaseConfig: {
            name: "db-prod"
        }
    }
}
```
