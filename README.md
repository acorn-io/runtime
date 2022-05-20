### WARNING: Consuming this project can expose you to chemicals, which are known to the State of California to cause cancer and birth defects or other reproductive harm.

### Quick Start 

##### Prerequisites 
To try out acorn you will need admin access to a Kubernetes cluster. Docker Desktop, Rancher Desktop, and K3s are all great options for trying out acorn for testing/development.

##### Install

Download the latest acorn [release](https://github.com/acorn-io/acorn/releases/latest) from the github.

Untar or unzip the downloaded artifact

```shell
#Linux or macOS
tar -xzvf <release>.tar.gz

#test binary (assume local directory)
./acorn 
```

*note*: On macOS systems, you will need to go to system preferences -> security & privacy -> General to allow the binary to run.

##### Build/Run First Acorn

Create a new `acorn.cue` in your working directory and add the following contents.
```cue
containers: {
	web: {
		image: "nginx"
		publish: "80/http"
		files: {
			// Simple index.html file
			"/usr/share/nginx/html/index.html": "<h1>My First Acorn!</h1>"
		}
	}
}
```
Save the file. What this file defines is a container called *web* based on the nginx container image on Dockerhub. It also declares that port 80 should be exposed and that it will expose an http protocol service. We are also customizing the `index.html` file as part of our packaging process. The contents of the file will be added during the build process.

Now you will need to build your acorn from this file by typing `acorn build .`. This will launch an acorn builder and development registry into your Kubernetes cluster and build the acorn image.

```shell
> acorn build .
[+] Building 2.8s (5/5) FINISHED
 => [internal] load .dockerignore                                                                                       0.0s
 => => transferring context: 2B                                                                                         0.0s
 ...
 => => pushing layers                                                                                                   0.0s
 => => pushing manifest for 127.0.0.1:5000/acorn/acorn:latest@sha256:ec773716b1d180ce4e343cdb4d84736107655401a3d411728  0.0s
346 / 55365718
60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3
```

Now we have a built acorn image identified by the sha (60d803258f7a...) at the end of our build command. To run our acorn app we need to run it.
```shell
> acorn run 60d803258f7a
little-dew
```

Our acorn has started and is named `little-dew`. 

To check the status of our app we can run the following. 
```shell
> acorn apps little-dew
NAME         IMAGE                                                              HEALTHY   UPTODATE   CREATED              ENDPOINTS                                   MESSAGE
little-dew   60d803258f7aa2680e4910c526485488949835728a2bc3519c09f1b6b3be1bb3   1         1          About a minute ago   http://web.little-dew.localhost => web:80   OK
```

In Chrome or Firefox browsers you can now open the URL listed under the endpoints column. 

There is a lot more you can do with an Acorn package. // TODO: see docs for more info. 


### Reference acorn.cue file
```cue
// The acorn file is a cue syntax. CUE is a superset of JSON, so all valid JSON is CUE.  In CUE you don't
// need to quote most keys, comments using // are supported, and trailing commas are optional.

// Definitions of containers to run. The key of will be used as the container name and must be a valid
// short DNS name.  The container(s) will be accesible through DNS by this short name. The keys have to be
// unique across containers, jobs, cronjobs, and routers.
containers: {
	// A definition of a container to run. The scale field below will determine how many containers from this
	// one definition.  At runtime each container will have a unique hostname.
	sample: {
		// The Docker/OCI image to run for this container. Either image or build must be specified
		image: "some-docker/image:v1.0.0"

		// Build the image to run from a Dockerfile. This can either be a path to where the Dockerfile
		// for example:
		//
		//    build: "./image"
		//
		// The above will is the same as {dockerfile: "./image/Dockerfile", context: "./image"}
		// is or full build definition like below.
		build: {
			// Build arguments to pass to the Docker build.
			args: {
				param1: "value"
			}
			// The context root of the Docker build relative to the root of the acorn build
			context: "./subdir"
			// Location of the Dockerfile to use relative to the root of the acorn build, not the context path
			dockerfile: "./subdir/otherdir/Dockerfile"
			// The multi-stage build target to build in the Dockerfile
			target: string | *""
		}

		// Override the ENTRYPOINT defined in the Dockerfile
		entrypoint: ["echo", "hello", "world"]

		// Override the CMD defined in the Dockerfile
		cmd: ["echo", "hello", "world"]

		// Environment variables to set on the running process
		env: {
			VAR1: "value"
			VAR2: "value"
		}

		// Override the WORKDIR defined in the Dockerfile
		workDir: "/tmp"

		// Allocate a TTY
		interactive: false

		// Ports to open to other containers in the existing app (no publically accesible)
		// Ports are of the format "[EXTERNAL:]INTERNAL[/PROTOCOL]". Acceptable protocols
		// are tcp, udp, http, https
		ports: [
			// TCP port 22
			22,

			// TCP port 2222 externally accessible as port 22
			"22:2222",

			// HTTP port 80
			"80/http",

			// UDP port 1234
			"1234/udp",
		]

		// Publish these ports externally (publically accesible) and internally. The format is the same as "ports"
		publish: [
			22,
			"22:2222",
			"80/http",
			"1234/udp",
		]

		// Files to inject into the running container.  The key must be the path of the file to create. The value
		// is any arbitrary string or bytes. Secrets values can be injected into files using the syntax
		// "secret://secret-name/data-key"
		files: {
			// Sample of a string value
			"/var/lib/www/index.html": "<h1>Hello World</h1>"

			// Sample of a byte value
			"/var/lib/www/1px.png":
				'\x89\x50\x4e\x47\x0d\x0a\x1a\x0a\x00\x00\x00\x0d' +
				'\x49\x48\x44\x52\x00\x00\x00\x01\x00\x00\x00\x01' +
				'\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00' +
				'\x0d\x49\x44\x41\x54\x08\x5b\x63\x48\xde\xda\xf5' +
				'\x1f\x00\x05\xc3\x02\xa2\x30\x41\xf9\xda\x00\x00' +
				'\x00\x00\x49\x45\x4e\x44\xae\x42\x60\x82'

			// Sample of a secret using the format "secret://secret-name/data-key"
			"/run/secret/password": "secret://sample-user/password"
		}

		// Mount a volume, secrets, or ephemeral storage a directory or secrets.
		// Secrets are referenced by the value "secret://secret-name". The secret data will become files
		// named the same as the keys.
		// Volumes are reference by "volume://name?class=className&size=10" where size is in gigabytes.
		dirs: {
			// Mount volume named "my-data"
			"/var/lib/data": "my-data"
			// Mount volume named "my-data" using storage class "local" of size 30G
			"/var/lib/data2": "volume://my-data?class=local&size=30"
			// Mount anonymous ephemeral storage
			"/var/lib/data3": "ephemeral://"
			// Mount named ephemeral storage. Named ephemeral storage can be used to share data between sidecars
			"/var/lib/data4": "ephemeral://my-eph"
			// Mount the secret named secret-name
			"/var/lib/data4": "secret://secret-name"
		}

		// Sidecars are containers that run colocated with the parent container and share the
		// same network and PID namespace.
		sidecars: {
			"sample-sidecar": {
				// All of the same fields of containers are supported here
			}
		}
	}
}

// Jobs are containers that will run at least once to completion and then not executed
// again.
jobs: {
	"sample-job": {
		// All of the same fields of containers are supported here
	}
}

// Volumes hold persistent data that can be mounted into containers
volumes: {
	"vol1": {
		// size of the requested volume in gigabytes
		size: 50
		// Valid access modes are "readWriteMany", "readWriteOnce", "readOnlyMany", "readWriteOncePod"
		accessModes: ["readWriteOnce"]
	}
}

secrets: {
	"generated-basic-auth": {
		// Represents a standard basic auth of username/password
		type: "basic"
		data: {
			// If not set a username will be generated
			username: string
			// If not set a password will be generated
			password: string
		}
	}
	"generated-tls": {
		type: "tls"
		params: {
			algorithm:   "rsa" | *"ecdsa"
			caSecret?:   string
			usage:       *"server" | "client"
			commonName?: string
			organization: [...string]
			sans: [...string]
			durationDays: int | *365
		}
		data: {
			"tls.crt"?: (string | bytes)
			"tls.key"?: (string | bytes)
			"ca.crt"?:  (string | bytes)
			"ca.key"?:  (string | bytes)
		}
	}
	"generated-from-job": {
		type: "generated"
		params: {
			job:    string
			format: *"text" | "json"
		}
	}
	"generated-from-template": {
		type: "template"
		data: {
			// Other secrets can be referenced using the syntax ${secret://secret-name/key}
			template: string
		}
	}
	"generated-token": {
		type: "token"
		params: {
			characters: string | *"bcdfghjklmnpqrstvwxz2456789"
			length:     (>=0 & <=256) | *54
		}
		data: {
			token?: string
		}
	}
	"pull-secret": {
		// Pull secrets are expected to be in this format
		type: "docker"
		data: {
			// Contents of ~/.docker/config.json
			".dockerconfigjson": "..."
		}
	}
}

// Schema of parameters to impact behavior at build or deploy time
params: {
	build: {
		"some-string": string | *"default value"
		"some-int": int | *5
		"some-complex": {
			key?: {
				key: string
			}
		}
	}
	deploy: {
		"some-string": string | *"default value"
		"some-int": int | *5
		"some-complex": {
			key?: {
				key: string
			}
		}
	}
}

// Arbitrary information that the acorn file author can embed so that if can be
// referenced somewhere else in the file. This is used mostly as a way to organize or better
// format your acorn file
data: {
	key: "value"
	complex: {
		key: "value"
	}
}
```
