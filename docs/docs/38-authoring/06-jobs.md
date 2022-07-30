---
title: Jobs
---

Jobs are containers that perform one off or scheduled tasks to support the application. Jobs are defined in their own top-level `jobs` section of the Acornfile. A job container will continue to run until it has successfully completed all operations once.

A Job has all the same fields as a container, with the exception of an optional `schedule` field.

## On update jobs

By default, jobs will run whenever the Acorn has been updated.

```cue
jobs: {
    "cluster-reconcile": {
        image: "registry.io/myorg/cluster-reconcile"
        env: {
            "CLUSTER_PASS": "secret://cluster-auth-token/token"
        }
    }
}
```

## Scheduled jobs

Jobs that need to be run on a schedule, like a backup job, must also define the schedule field.

```cue
jobs: {
    "db-backup": {
        image: "registry.io/myorg/db-backup"
        env: {
            "BACKUP_USER": "secret://backup-user-creds/username"
            "BACKUP_PASS": "secret://backup-user-creds/password"
        }
        command: ["/scripts/backup"]
        schedule: "@hourly"
    }
}
```

The `schedule` key makes this a cron based job. The `schedule` field must be a valid crontab format entry. Meaning it can use standard `* * * * *` format or @[interval] crontab shorthand.
