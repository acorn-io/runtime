---
title: Acornfile
---

## Root

The root level elements are,
[args](#args)
[containers](#containers),
[jobs](#jobs),
[volumes](#volumes),
[secrets](#secrets),
[acorns](#acorns),
and [localData](#localData).

[containers](#containers),
[jobs](#jobs), and
[acorns](#acorns) are all maps where the keys must be unique across all types. For example, it is
not possible to have a container named `foo` and a job named `foo`, they will conflict and fail. Additional
the keys could be using in a DNS name so the keys must only contain the characters `a-z`, `0-9` and `-`.

```acorn
// User configurable values that can be changed at build or run time.
args: {
}

// Definition of containers to run
containers: {
}

// Defintion of jobs to run
jobs: {
}

// Definition of volumes that this acorn needs to run
volumes: {
}

// Definition of secrets that this acorn needs to run
secrets: {
}

// Definition of Acorns to run
acorns: {
}

// Arbitrary information that can be embedded to help render this Acornfile
localData: {
}
```
## containers

`containers` defines the templates of containers to be ran. Depending on the 
scale parameter 1 or more containers can be created from each template (including their [sidecars](#sidecars)).

```acorn
containers: web: {
	image: "nginx"
	ports: publish: "80/http"
}
```

### dirs, directories

`dirs` configures one or more volumes to be mounted to the specified folder.  The `dirs` parameter is a
map structure where the key is the folder name in the container and the value is the referenced volume. Refer
to the [volumes](#volumes) section for more information on volume types.

```acorn
containers: default: {
	image: "nginx"
	dirs: {
		// A volume named "volume-name" will be mounted at /var/tmp
		"/var/tmp": "volume-name"
		
		// A volume named "volume-name" will be mounted at /var/tmp-full with the size of 20G and an
		// access mode of readWriteMany
		"/var/tmp-full": "volume://volume-name?size=20G,accessMode=readWriteMany"
		
		// An ephemeral volume will be created and mounted to "/var/tmp-ephemeral"
		"/var/tmp-ephemeral": "ephemeral://"
		
		// An ephemeral volume named "shared" will be created and mounted to "/var/tmp-ephemeral"
		"/var/tmp-ephemeral-named": "ephemeral://shared"
		
		// A folder will be created at /var/tmp-secret/ where the filenames are the
		// key names of the secret "sec-name" and the contents of each file is the corresponding
		// secret data value
		"/var/tmp-secret": "secret://sec-nam"
		
		// The local folder ./www will be copied during build into the container image
		// as /var/www.  If running in dev mode the directory will be syncronized live with
		// changes.  Local folders must start with "./".
		"/var/www": "./www"
	}
	sidecars: sidecar: {
		image: "ubuntu"
		dirs: {
			// An ephemeral volume named "shared" will be mounted to /var/tmp with the contents of
			// the volume shared with the main containers /var/tmp-ephemeral-named folder
            "/var/tmp": "ephemeral://shared"
		}
	}
```


### files

`files` will create files in the container with content from the Acornfile or the value of a secret. The
`files` parameter is a map structure with the key being the file name and the value being the text of the file
or a reference to a secret value. The default mode for files is `0644` unless the file ends with `.sh` or contains
`/bin/` or `/sbin/`.  In those situations the mode will be `0755`.
```acorn
containers: default: {
	image: "nginx"
	files: {
		// A file named /var/tmp/file.txt will be created with mode 0644
		"/var/tmp/file.txt": "some file contents"
		
		// A file named /run/secret/password will be created with mode 0400 with the
		// contents of from the secret named "sec-name" and the value of the data
		// key named "key".
		"/run/secret/password": "secret://sec-name/key?mode=0400"
		
		// By default if a secret value changes the application will be restarted.
		// the following example will cause the container to not be restarted when
		// the secret value changes, but instead the container is dynamically updated
		"/run/secret/password-reload": "secret://sec-name/key?onchange=no-action"
		
		// A file /var/tmp/other.txt will be created with a custom mode value "0600"
		"/var/tmp/other.txt": {
			content: "file content"
			mode: "0600"
		}
		
	}
}
```
### image
`image` refers to the OCI (Docker) image to run for this container.
```acorn
containers: web: {
	image: "nginx"
}
```
### build
`build` contains the information need to build an OCI image to run for this container
```acorn
containers: build1: {
	// Build the Docker image using the context dir "." and the "./Dockerfile".
	build: "."
}

containers: build2: {
	build: {
		// Build using the context dir "./subdir"
		context: "./subdir"
		// Build using the "./subdir/Dockerfile"
		dockerfile: "./subdir/Dockerfile"
		// Build with the multi-stage target named "multistage-target"
		target: "multistage-target"
		// Pass the following build arguements to the dockerfile
		buildArgs: {
			"arg1": "value1"
			"arg2": "value2"
		}
	}
}
```
### command, cmd
`command` will overwrite the `CMD` value set in the Dockerfile for the running container
```acorn
containers: arg1: {
	image: "nginx"
	// This command will be parsed as a shell expression and turned into an array and ran
	cmd: #"/bin/sh -c "echo hi""#
}

containers: arg2: {
	image: "nginx"
	// The following will not be parsed and will be ran as defined.
	cmd: ["/bin/sh", "-c", "echo hi"]
}

```
### entrypoint
`entrypoint` will overwrite the `ENTRYPOINT` value set in the Dockerfile for this running container
```acorn
containers: arg1: {
	image: "nginx"
	// This command will be parsed as a shell expression and turned into an array and ran
	entrypoint: #"/bin/sh -c "echo hi""#
}

containers: arg2: {
	image: "nginx"
	// The following will not be parsed and will be ran as defined.
	entrypoint: ["/bin/sh", "-c", "echo hi"]
}
```
### env, environment
`env` will set environment variables on the defined container.  The value of the environment variable
may be static text or a value from a secret.
```acorn
containers: env1: {
	image: "nginx"
	env: [
	    // An environment variable of name "NAME" and value "VALUE" will be set
		"NAME=VALUE",

	    // An environment variable of name "SECRET" and value of the key "key" in the
	    // secret named "sec-name" will be set. When this secret changes the container
	    // will not be restarted.
		"SECRET=secret://sec-name/key?onchange=no-action"
	]
}

containers: env1: {
	image: "nginx"
	// The same configuration as above but in map form
	env: [
		NAME: "VALUE"
		SECRET: "secret://sec-name/key?onchange=no-action"
	]
}
```
### workDir, workingDir
`workDir` sets the current working directory of the running process defined in `cmd` and `entrypoint`
```acorn
containers: env1: {
	image: "nginx"
	command: "ls"
	// Run the command "ls", as defined above, in the directory "/tmp"
	workDir: "/tmp"
}
```

### dependsOn
`dependsOn` will prevent a container from being created and/or updated until all of
it's dependencies are considered ready. Ready service are considered ready as soon as
their [ready probe](#probes-probe) passes.  If there is no ready probe it is as soon
as the containers have started.
```acorn
containers: web: {
	image: "nginx"
	dependsOn: ["db"]
}
containers: db: {
	// ...
	image: "mariadb"
}
```
### ports
`ports` defines which ports are available on the container and the default level of access. Ports
are defined with three different access modes: internal, expose, publish. Internal ports are only available
to the containers within an Acorn. Expose(d) ports are available to services within the cluster. And
publish ports are available publically outside the cluster. The access mode defined in the Acornfile is
just the default behavior and can be changed at deploy time.
```acorn
containers: web: {
	image: "nginx"
	
	// Define internal TCP port 80 available internally as DNS web:80
	ports: 80
	
	// Define internal HTTP port 80 available internally as DNS web:80
	// Valid protocols are tcp, udp, and http
	ports: "80/http"
	
	// Define internal HTTP port 80 that maps to the container port 8080
	// available internally as DNS web:80
	ports: "80:8080/http"
	
	// Define internal TCP port 80 that maps to the container port 8080
	// available internally as DNS web:80
	ports: "80:8080"
	
	// Define internal TCP port 80 that maps to the container port 8080
	// available internally as DNS web-alias:80
	ports: "web-alias:80:8080"
	
	// The similar ports as above but just in a list
	ports: [
		80,
	    "80/http",
	    "80:8080/http",
	]
	
	ports: {
		// Define publically accessible HTTP port 80 that maps to the container port 8080
	    // available publically as a DNS assigned at runtime
		publish: ["80:8080/http"]
		
		// Define cluster accessible HTTP port 80 that maps to the container port 8080
	    // available publically as a DNS assigned at runtime
		expose: ["80:8080/http"]
		
	    // Define internal HTTP port 80 that maps to the container port 8080
	    // available internally as DNS web:80
		internal: ["80:8080/http"]
	}
	
    // Define publically accessible HTTP port 80 that maps to the container port 8080
    // available publically as a DNS assigned at runtime
	ports: publish: "80:8080/http"
}
```

### probes, probe
`probes` configure probes that can signal when the container is ready, alive, and started. There are
three probe types: `readiness`, `liveness`, and `startup`. `readiness` probes indicate when an application
is available to handle requests. Ports will not be accessible until the `readiness` probe passes. `liveness`
probes indicate if a running process is healthy. If the `liveness` probe fails, the container will be deleted
and restarted. `startup` probe indicates that the container has started for the first time.

```acorn
containers: web: {
	image: "nginx"
	
	// Configure readiness probe to run probe.sh
	probe: "probe.sh"
	
	// Configure readiness probe to look for a HTTP 200 response from localhost port 80
	probe: "http://localhost:80"
	
	// Configure readiness probe to connect to TCP port 1234
	probe: "tcp://localhost:1234"
	
	probes: {
		"readiness": {
            // Configure a HTTP readiness probe
            http: {
                url: "http://localhost:80"
                headers: {
                    "X-SOMETHING": "some-value"
                }
            }
            // Below are the default values for the following parameters
            initialDelaySeconds: 0
            timeoutSeconds: 1
            periodSeconds: 10
            successThreshold: 1
            failureThreshold: 3
		}
		"liveness": {
            // Configure an Exec liveness probe
            exec: {
                command: ["probe.sh"]
            }
		}
		"startup": {
            // Configure a TCP startup liveness probe
            tcp: {
            	url: "tcp://localhost:1234"
            }
		}
	}
}

```
### scale
`scale` configures the number of container replicas based on this configuration that should
be ran.

```acorn
containers: web: {
	image: "nginx"
	scale: 2
}
```

### sidecars
`sidecars` are containers that run colocated with the parent container and share the same network
address. Sidecars accept all the same parameters as a container and one additional parameter `init`
```acorn
containers: web: {
	image: "nginx"
	sidecars: sidecar: {
		image: "someother-image"
	}
}
```
#### init

`init` tells the container runtime that this `sidecar` must be ran first on startup and the main
container will not run until this container is done

```acorn
containers: web: {
    image: "nginx"
    dirs: "/run/startup/": "ephemeral://startup-info"
    sidecars: "stage-data": {
    	// This sidecar will run first and only when it exits with exit code 0 will the 
    	// parent "web" container start
    	init: true
    	image: "ubuntu"
        dirs: "/run/startup/": "ephemeral://startup-info"
        command: "./stage-data-to /run/startup"
    }
}
```

## jobs
`jobs` are containers that are run once to completion. If the configuration of the job changes, the will
be ran once again.  All fields that apply to [containers](#containers) also apply to
jobs.

```acorn
jobs: "setup-volume": {
	image: "my-app"
	command: "init-data.sh"
	dirs: "/mnt/data": "data"
}
```
### schedule
`schedule` field will configure your job to run on a cron schedule. The format is the standard cron format.

```
 ┌───────────── minute (0 - 59)
 │ ┌───────────── hour (0 - 23)
 │ │ ┌───────────── day of the month (1 - 31)
 │ │ │ ┌───────────── month (1 - 12)
 │ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday;
 │ │ │ │ │                                   7 is also Sunday on some systems)
 │ │ │ │ │                                   OR sun, mon, tue, wed, thu, fri, sat
 │ │ │ │ │
 * * * * *
```
The following shorthand syntaxes are supported

| Entry                   | 	Description	                                            | Equivalent to |
|-------------------------|------------------------------------------------------------|---------------|
| @yearly (or @annually)  | Run once a year at midnight of 1 January	                | 0 0 1 1 *     |
| @monthly	               | Run once a month at midnight of the first day of the month | 0 0 1 * *     |
| @weekly	               | Run once a week at midnight on Sunday morning	            | 0 0 * * 0     |
| @daily (or @midnight)   | Run once a day at midnight	                                | 0 0 * * *     |
| @hourly	               | Run once an hour at the beginning of the hour	            | 0 * * * *     |

## volumes
`volumes` store persistent data that can be mounted by containers
```acorn
container: db: {
	image: "mariadb"
	dirs: "/var/lib/mysql": "data"
}
volumes: data: {
	size: "100G"
	accessModes: "readWriteOnce"
	class: "default"
}
```
### size
`size` configures the default size of the volume to be created.  At deploy-time this value can be
overwritten.

```acorn
volumes: data: {
	// All numbers are assumed to be gigabytes
	size: 100

	// The following suffixes are understood
    // 2^x  - Ki | Mi | Gi | Ti | Pi | Ei
    // 10^x - m | k | M | G | T | P | E
	size: "10G"
}
```
### class
`class` refers to the `storageclass` within kubernetes. 
```acorn
volumes: data: {
        // either "default" or a storageclass from `kubectl get sc`
	class: "longhorn"
}
```
### accessModes
`accessModes` configures how a volume can be shared among containers.

```acorn
volumes: data: {
	accessModes: [
		// Only usable by containers on the same node
		"readWriteOnce",
		// Usable by containers across many nodes
		"readWriteMany",
		// Usable by containers across many nodes but read only
		"readOnlyMany",
	]
}
```

## secrets

`secrets` store sensitive data that should be encrypted as rest.

```acorn
secrets: "my-secret": {
    type: "opaque"
    data: {
        key1: ""
        key2: ""
    }
}
```

### type
The common pattern in Acorn is for secrets to be generated if not supplied. `type`
specifies how the secret can be generated. Refer to [the secrets documentation](../38-authoring/05-secrets.md) for
descriptions of the different secret types and how they are used.

```acorn
secrets: "a-token": {
	// Valid types are "opaque", "token", "basic", "generated", and "template"
	type: "opaque"
}
```

### params
`params` are used to configure the behavior of the secrets generation for different types.
Refer to [the secrets documentation](../38-authoring/05-secrets.md) for
descriptions of the different secret types and how their parameters.
```acorn
secrets: "my-token": {
    type: "token"
    params: {
        length: 32
        characters: "abcdedfhifj01234567890"
    }
}
```
### data
`data` defines the keys and non-senstive values that will be used by the secret.
Refer to [the secrets documentation](../38-authoring/05-secrets.md) for
descriptions of the different secret types and how to use data keys and values.

```acorn
secrets: {
    "my-template": {
        type: "template"
        data: {
            template: """
            a ${secret://my-secret-data/key} value
            """
        }
    }
    "my-secret-data": {
        type: "opaque"
        data: {
            key: "value"
        }
    }
}
```

## acorns
### image
### build
### profiles
### deployArgs
### ports
### secrets
### volumes
### environment, env
### links

## args
## localData
