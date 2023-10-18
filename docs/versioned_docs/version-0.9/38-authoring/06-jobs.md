---
title: Jobs
---

Jobs are containers that perform one-off or scheduled tasks to support the application. Jobs are defined in their own top-level `jobs` section of the Acornfile. A job container will continue to run until it has successfully completed all operations once.

A Job has all the same fields as a container, with the exception of an optional `schedule` and `events` field.

## Scheduled jobs

Jobs that need to be run on a schedule, like a backup job, must also define the schedule field.

```acorn
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

## Events

Acorn supports four events that can trigger a job to run: `create`, `update`, `stop`, and `delete`. By default jobs will run on create and update. To change this behavior, use the `events` field.

The `create` event will run the job when the app is created, or when the job is first added to the Acornfile.

The `update` event will run the job when the app is updated or started from stop.

The `stop` event will run the job when the app is stopped.

The `delete` event will run the job when the app is deleted. The job will run, and must complete successfully, before the remaining containers are deleted in that Acorn app. If the job fails, the app will not be deleted. To skip the job, use the [`--ignore-cleanup`](100-reference/01-command-line/acorn_rm.md#options) flag.

```acorn
jobs: {
    "cluster-reconcile": {
        image: "registry.io/myorg/cluster-manager"
        env: {
            "CLUSTER_PASS": "secret://cluster-auth-token/token"
        }
        events: ["create", "update", "stop", "delete"]
        entrypoint: ["/lc.sh"]
        files: {
            "/lc.sh": """
              #!/bin/sh
              if if [ "${ACORN_EVENT}" = "create" ]; then
                echo "Create event"
                exit 0
              elif [ "${ACORN_EVENT}" = "update" ]; then
                echo "Update event"
                exit 0
              elif [ "${ACORN_EVENT}" = "stop" ]; then
                echo "Stop event"
                exit 0
              elif [ "${ACORN_EVENT}" = "delete" ]; then
                echo "Delete event"
                exit 0
              else
                echo "Unknown event"
                exit 1
              fi
            """
        }
    }
}
```
