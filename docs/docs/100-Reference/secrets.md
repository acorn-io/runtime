---
title: Secrets
---


## Types

### Token

The token secret type is used to generate a random token string that can be used for a password or credential.

```cue
// Acorn.cue
...
containers: {
    app: {
        ...
        files: {
            "/config/secret_token": "secret://my-app-token/token"
        }
    }
}
...
secrets: {
    ...
    "my-app-token": {
        type: "token"
        params: {
            length: 32
            chars: "1234567890"
        }
    }
    ...
}
...
```

The above generates a token 32 characters in length using only digits 0-9.

### Generated

Generated secrets are values obtained from running a job defined in the Acornfile file. The job must write the output to `/run/secrets/output` in order for it to be placed into the secret. The data will be available via the secrets `content` key.

```cue
containers: {
    app: {
        image: "httpd:2"
        files: {
            "/etc/httpd/htpasswd": "secret://htpasswd-file/content"
        }
    }
}
jobs: {
    "htpasswd-create": {
        env: {
            "USER": "secret://user-creds/username"
            "PASS": "secret://user-creds/password"
        }
        entrypoint: "/bin/sh -c"
        image:      "httpd:2"
        // Output of a generated secret needs to be placed in the file /run/secrets/output.
        cmd: ["htpasswd -Bbc /run/secrets/output $USER $PASS"]
    }
}
secrets: {
    "user-creds": {
        type: "basic"
    }
    "htpasswd-file": {
        type: "generated"
        params: {
            job: "htpasswd-create"
        }
    }
}
```

### Basic

This secret type is used to store a username and password. If no values are provided, Acorn will generate random strings to be used.

```cue
args: deploy: username: string | *""
args: deploy: password: string | *""
secrets: {
    // Always generate random user and password
    "app-creds": {
        type: "basic"
        data: {}
    }
    // Always generate a password for the user root
    "root-creds": {
        type: "basic"
        data: {
            username: "root"
        }
    }
    // Allow user to pass username and/or password. Otherwise generate
    "user-provided-creds": {
        type: "basic"
        data: {
            username: "\(args.deploy.username)"
            password: "\(args.deploy.password)"
        }
    }
}
```

### Opaque

Opaque secrets are for sensitive bits of data without a specific structure. One example usecase is to pass in ssh keys from a user.

```cue
containers: {
    git: {
        image: "my-git"
        dirs: {
            "/home/user/.ssh": "secret://my-keys"
        }
    }
}
secrets: {
    "user-provided-ssh-keys": type: "opaque"
}
```

Now assuming a user has a pre-created secret with keys called `my-keys` the Acorn be launched like so.

```shell
> acorn run -s my-keys:user-provided-ssh-keys [MY-APP-IMAGE]
morning-pine
```

### Template

The template type makes it easy to provide config files and scripts with string interpolated values.

```cue
args: deploy: setting: string| *"default"
containers: {
    app: {
        files:{
            "/etc/my.cfg": "secret://my-template-config/template"
        }
    }
}
secrets: {
    "my-template-config": {
        type: "template"
        data: {
            template: """
            setting-a: \(args.deploy.setting)
            """
        }
    }
}
```

You can accomplish more complex templating using some cue functions. To render YAML:

```cue
import "encoding/yaml"

containers: {
    app: {
        files: {
            "/etc/config.yaml": "secrets://yaml-template/template"
        }
    }
}
secrets: {
    "yaml-template": {
        type: "template"
        data: {template: yaml.Marshal(localData.config)}
    }
}
localData: {
    config: {
        toplevel: {
            subkey: "value"
            subkey0: "other value"
        }
    }
}
```

Notice that we are importing a function to handle the YAML marshaling for us.
