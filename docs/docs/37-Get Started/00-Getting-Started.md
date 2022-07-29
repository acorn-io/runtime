---
title: Getting Started
---

In this walkthrough you will build a Python web app, package it up and deploy it as an Acorn App.  
The app will interact with Redis and Postgres, which both will be packaged along with the web app in a single Acorn Image.

> The guide makes use of Python, Redis and Postgres here, but you don't need to be familiar with those technologies, as the examples should be understandable without preliminary knowledge in those.

## Prerequisites

To run this example, you will need to have the Acorn CLI installed and administrative access to a Kubernetes cluster.
Here you can find some documentation on how to get there:

- [Acorn CLI](/30-Installation/01-installing.md)
- Kubernetes: Acorn works well with local development instances like provided by [Rancher Desktop](https://docs.rancherdesktop.io/getting-started/installation), [K3s](https://rancher.com/docs/k3s/latest/en/quick-start/), [k3d](https://k3d.io/v5.4.4/#installation) or [Docker Desktop](https://www.docker.com/get-started/). It also works with any other Kubernetes distribution, for example a managed instance hosted in a major cloud provider or a cluster provided by your work environment.

## 0. Prepare the cluster

Installing the Acorn server-side components into your cluster is as easy as running

```bash
acorn install
```

> Note: Installing Acorn into a Kubernetes cluster requires cluster-admin privileges. Please see our [architecture overview](/60-Architecture/01-ten-thousand-foot-view.md) to learn what components will be deployed.

## 1. Create your App

Let's start by creating a few files that compose our Python web app.

```bash
# Create a directory structure for our web app
mkdir acorn-test-app

# Change into the root of the new directory tree
cd acorn-test-app
```

### The Web App

Now create the basic Python web app in a file called `app.py`, which stores a visitor count in the Redis Cache and fetches a list of items from a Postgres Database.

```python title="acorn-test-app/app.py"
import logging as log
import os

import psycopg2
import redis
from flask import Flask, render_template_string

# HTML Jinja2 Template which will be shown in the browser
page_template = '''
        <div style="margin: auto; text-align: center;">
        <h1>{{ welcome_text }}</h1><br>
        You're visitor #{{ visitors }} to learn what squirrels love the most:<br>
        <ul>
            {%- for food in foods %}
            <li>{{ food }}</li>
            {%- endfor %}
        </ul>
        </div>
        '''

# Defining the Flask Web App
app = Flask(__name__)
cache = redis.StrictRedis(host='cache', port=6379)


# The website root will show the page_template rendered with
# - visitor count fetched from Redis Cache
# - list of food fetched from Postgres DB
# - welcome text passed in as environment variable
@app.route('/')
def root():
    visitors = cache_get_visitor_count()
    food = db_get_squirrel_food()

    return render_template_string(page_template, visitors=visitors, foods=food, welcome_text=os.getenv("WELCOME", "Hey Acorn user!"))


# Fetch the squirrel food from the Postgres database
def db_get_squirrel_food():
    conn = psycopg2.connect(
        host="db",
        database="acorn",
        user=os.environ['PG_USER'],
        password=os.environ['PG_PASS'],
    )

    cur = conn.cursor()
    cur.execute("SELECT food FROM squirrel_food;")

    return [x[0] for x in cur.fetchall()]  # Return the list of food items


# Increment the visitor count in the Redis cache and return the new value
def cache_get_visitor_count():
    return cache.incr('visitors')
```

### Python Requirements

Alongside the Python code, we need the list of dependencies to install. Save the following in the file `requirements.txt`:

```txt title="acorn-test-app/requirements.txt"
flask
psycopg2-binary
redis
```

### Dockerfile

Now we have all the code and want to bundle it up in a Docker container.
Create the `Dockerfile` with the following content:

```dockerfile title="acorn-test-app/Dockerfile"
FROM python:3-alpine
WORKDIR /app
ENV FLASK_APP=app.py
ENV FLASK_RUN_HOST=0.0.0.0
RUN apk add --no-cache gcc musl-dev linux-headers
COPY . .
RUN pip install -r requirements.txt
EXPOSE 5000
CMD ["flask", "run"]
```

## 2. Author your Acornfile

> Don't get scared by the wall of configuration below. It's demonstrating a lot of features that you could totally live without for a simple app.

```cue title="acorn-test-app/Acornfile"
args: {
  // Configure your personal welcome text
  welcome: "Hello Acorn User!"
}
containers: {
  app: {
    build: "."
    env: {
      "PG_USER": "postgres"
      "PG_PASS": "secret://quickstart-pg-pass/token"
      "WELCOME": args.welcome
    }
    dependsOn: [
      "db",
      "cache"
    ]
    ports: publish: "5000/http"
  }
  cache: {
    image: "redis:alpine"
    ports: "6379/tcp"
  }
  db: {
    image: "postgres:alpine"
    env: {
      "POSTGRES_DB": "acorn"
      "POSTGRES_PASSWORD": "secret://quickstart-pg-pass/token"
    }
    dirs: {
      if !args.dev {
        "/var/lib/postgresql/data": "volume://pgdata"
      }
    }
    files: {
      "/docker-entrypoint-initdb.d/00-init.sql": "CREATE TABLE squirrel_food (food text);"
      "/docker-entrypoint-initdb.d/01-food.sql": std.join([for food in localData.food {"INSERT INTO squirrel_food VALUES ('\(food)');"}], "\n")
    }
    ports: "5432/tcp"
  }
}
localData: {
  food: [
    "acorns",
    "hazelnuts",
    "walnuts"
  ]
}
volumes: {
  if !args.dev {
    "pgdata": {
      accessModes: "readWriteOnce"
    }
  }
}
secrets: {
  "quickstart-pg-pass": {
      type: "token"
  }
}
```

### Explaining the Acornfile

- `args` section: describes a set of arguments that can be passed in by the user of this Acorn Image
  - A help text will be auto-generated using the comment just above the arg:

    ```bash
    $ acorn run . --help
    Usage of Acornfile:
          --welcome string   Configure your personal welcome text
    ```

- `containers` section: describes the set of containers your Acorn App consists of
  - Note: `app`, `db` and `cache` are custom names of your containers
  - `app` - Our Python Flask App
    - `build`: build from Dockerfile that we created
    - `env`: environment variables, statically defined, referencing a secret or referencing an Acorn argument
    - `dependsOn`: dependencies that have to be up and running before the app is started (here it is waiting for the database and the cache to be running)
    - `ports`: using the `publish` type, we expose the app inside the cluster but also outside of it using an auto-generated ingress resource (more on this later <!-- TODO: add link -->)
  - `cache` - Redis
    - `image`: existing OCI/Docker image to use (here: from DockerHub library)
    - `ports`: no type defined, defaults to `internal`, which makes it available to the other containers in this Acorn App
  - `db` - Postgres Database Server
    - `image`, `env`,`ports`: nothing new here
    - `dirs`: Directories to mount into the container filesystem
      - `if !args.dev`: only apply, if built-in development mode is **not** active (more on the development mode later <!-- TODO: add link -->)
      - `volume://pgdata`: references a volume defined in the top-level `volumes` section in the Acornfile. Also supports other references. <!-- TODO: add link -->
    - `files`: Similar to `dirs` but only for files. Additionally, content can be created in-line and even utilizing generating functions.
  - `localData`: Set of variables for this Acorn App
    - `food`: Custom variable, defining a list of food which is accessed in `containers.db.volumes` to pre-fill the database.
  - `volumes`: (persistent) data volumes to be used by any container in the Acorn App
    - `pgdata` custom volume name, referenced in `containers.db.dirs`
      - `accessModes`: (list of) modes to allow access to this volume
  - `secrets`: set of secrets that can be auto-generated and used by any container in the Acorn App
    - `quickstart-pg-pass`: custom secret name, referenced by `containers.app.env` and `containers.db.env`
      - `type`: There are several secret types <!-- TODO: add link-->. Here, a token (random string) will be generated for you at runtime.

## 3. Run your Acorn App

### Normal Mode

To run your Acorn App in "normal" operations mode, just run

```bash
acorn run .
```

or customize the welcome text argument via

```bash
acorn run . --welcome "Let's Get Started"
```

### Development Mode

<!-- TODO:
- `acorn run -i .`
  - hot-reloading
  - Q: Does this use the dev profile?
-->

## 5. Access your App

Due to the configuration `ports: publish: "5000/http"` under `containers.app`, our web app will be exposed outside of our Kubernetes cluster using the cluster's ingress controller.
Checkout the running apps via

```bash
acorn apps
```

Assuming that your Acorn App instance is called, `awesome-acorn`, this could look like this:

```bash
$ acorn apps
NAME         IMAGE          HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS                                             MESSAGE
awesome-acorn   2d73c8a0493f   3         3            121m ago   http://app.awesome-acorn.local.on-acorn.io => app:5000   OK
```

You probably already noticed the link right there in the `ENDPOINTS` column. It will take you to your Python Flask App.

<!-- FIXME: do we need a note on adding a port to the ingress controller here? -->

## 4. Update the Acornfile and push the changes to the running App

When not using the development mode, your typical deployment cycle involves at least building the image and deploying it (optionally pushing it to a registry in between).
These steps can be consolidated into a single command:

```bash
# Assuming that your Acorn App instance is called "awesome-acorn"
acorn update --image $(acorn build .) awesome-acorn
```

## 5. Build and Push your Acorn Image

Ready to release your Acorn App into the wild?
Let's package it up in a single Acorn Image and distribute it via an OCI registry (you could use DockerHub for that):

```bash
# Login into your OCI registry, if needed (interactive)
acorn login my.registry.com

# Build the Acorn Image and tag it to your liking
acorn build -t my.registry.com/acorn/getting-started:v0.0.1 .

# Push the newly built Acorn Image
acorn push my.registry.com/acorn/getting-started:v0.0.1
```

Now, everyone else can run your Acorn Image via

```bash
acorn run my.registry.com/acorn/getting-started:v0.0.1
```

## Play around with it

> Again, assuming that your deployed Acorn App is called `awesome-acorn`

### Execute a command inside the running container

The following will first let you choose the container you want to execute the command in and then run it.
You can also specify the container name upfront.

```bash
acorn exec awesome-acorn env
```

or: get an interactive shell inside a container via

```bash
acorn exec -i awesome-acorn sh
```

### Reveal the auto-generated database secret

```bash
# List all Acorn Secrets
$ acorn secrets
ALIAS                                  NAME                       TYPE      KEYS      CREATED
awesome-acorn.quickstart-pg-pass          quickstart-pg-pass-sqlv9   token     [token]   139m ago

# Reveal the one for the current app
$ acorn secret expose awesome-acorn.quickstart-pg-pass
NAME                       TYPE      KEY       VALUE
quickstart-pg-pass-sqlv9   token     token     mssl8692sk47tfklx9bqnqflw7pqrk2ldb6cd9tckjlttpk4vsvpvl
```

### Start and Stop your App

That's easy!

```bash
acorn stop awesome-acorn
```

and

```bash
acorn start awesome-acorn
```

### Remove your app

If you're done with the app, wipe it from the cluster via

```bash
acorn rm awesome-acorn
```

> Dangerous Pro-Tip: to remove **all** Acorn Apps at once: `acorn rm $(acorn apps -qa)`)

## What's next?

<!-- TODO:- Checkout some other sample Acorns -->
- [Explore all the other awesome Acorn commands](/100-Reference/01-command-line/acorn.md)
- [Read through the Acornfile reference](../100-Reference/03-Acornfile.md)
- [Have a look what makes up Acorn](/60-Architecture/01-ten-thousand-foot-view.md)
- Just continue reading on the next pages!
    Next, try the Sample apps with Compose
    Explore the full list of Compose commands
    Compose configuration file reference
    To learn more about volumes and bind mounts, see Manage data in Docker
