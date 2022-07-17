---
title: Managing Secrets
---

## Overview

Acorn Apps have secrets defined in the Acornfile file under the `secrets` key. Before running an Acorn App you can see which secrets are defined by passing the `--help` flag.

```shell
> acorn run [MY-APP-IMAGE] --help
Volumes:   mysql-data-0, mysql-backup-vol
Secrets:   backup-user-credentials, create-backup-user, user-provided-data, mariadb-0-client-config, mariadb-0-mysqld-config, mariadb-0-galera-config, root-credentials, db-user-credentials
Container: mariadb-0
Ports:     mariadb-0:3306/tcp

      --backup-schedule string         Backup Schedule
      --boot-strap-index int           Set server to boot strap a new cluster. Default (0)
      --cluster-name string            Galera: cluster name
      --custom-mariadb-config string   User provided MariaDB config
      --db-name string                 Specify the name of the database to create. Default(acorn)
      --db-user-name string            Specify the username of db user
      --force-recover                  When recovering the cluster this will force safe_to_bootstrap in grastate.dat for the bootStrapIndex node.
      --recovery                       Run cluster into recovery mode.
      --replicas int                   Number of nodes to run in the galera cluster. Default (1)
      --restore-from-backup string     Restore from Backup. Takes a backup file name
```

The secrets are listed out in the `Secrets` section.

## Listing secrets

If you would like to see which secrets are being used by your running Acorn Apps, you can use the acorn secret command.

```shell
> acorn secrets
NAME                            TYPE                        KEYS                  CREATED
backup-script-stzjk             secrets.acorn.io/template   [template]            18h ago
backup-user-credentials-sthvd   kubernetes.io/basic-auth    [password username]   18h ago
create-backup-user-xfxjp        secrets.acorn.io/template   [template]            18h ago
db-user-credentials-dvbsq       kubernetes.io/basic-auth    [password username]   18h ago
mariadb-0-client-config-hwvkv   secrets.acorn.io/template   [template]            18h ago
mariadb-0-galera-config-8kvxt   secrets.acorn.io/template   [template]            18h ago
mariadb-0-mysqld-config-m86m4   secrets.acorn.io/template   [template]            18h ago
root-credentials-vtw86          kubernetes.io/basic-auth    [password username]   18h ago
user-provided-data-67nwz        Opaque                      []                    18h ago
```

## Getting the secret contents

To see the values in the secret you can use the expose command.

```shell
> acorn secret expose db-user-credentials-dvbsq
db-user-credentials-dvbsq   kubernetes.io/basic-auth   password   m9r49sjkq58d5md4
db-user-credentials-dvbsq   kubernetes.io/basic-auth   username   hg4r98lh
```

## Using your own secret to pass data to the app

If you would like to pass your own credentials or secret information into the Acorn App, you can bind a secret in at runtime.

```shell
# Create the secret
> acorn secret create --data username=user0 --data password=supersecret1 my-app-secret-creds
my-app-secret-creds

# Bind into an app
> acorn run -s my-app-secret-creds:db-user-credentials [MY-APP-IMAGE]
morning-pine
```

In the above example the values from my-app-secret-creds will be used instead of the one created by the app.
