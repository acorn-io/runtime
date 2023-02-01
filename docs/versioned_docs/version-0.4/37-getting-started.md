---
title: Getting Started
---

In this walk through you will build a Python web app, package it up and deploy it as an Acorn app.
The app will interact with Redis and Postgres, which both will be packaged along with the web app in a single Acorn image.

> The guide makes use of Python, Redis and Postgres here, but you don't need to be familiar with those technologies, as the examples should be understandable without preliminary knowledge in those.

## Prerequisites

To run this example, you will need to have the Acorn CLI installed and administrative access to a Kubernetes cluster.
Here you can find some documentation on how to get there:

- [Acorn CLI](30-installation/01-installing.md)
- Access to a Kubernetes cluster through `kubectl` from your CLI. Some great options for local development are [Rancher Desktop](https://docs.rancherdesktop.io/getting-started/installation), [K3s](https://rancher.com/docs/k3s/latest/en/quick-start/), [k3d](https://k3d.io/v5.4.4/#installation), or [Docker Desktop](https://www.docker.com/get-started/). It also works with any other Kubernetes distribution, for example a managed instance hosted in a major cloud provider or a cluster provided by your work environment.

## Step 1. Prepare the cluster

Installing the Acorn server-side components into your cluster is as easy as running

```bash
acorn install
```

> Note: Installing Acorn into a Kubernetes cluster requires cluster-admin privileges. Please see our [architecture overview](60-architecture/01-ten-thousand-foot-view.md) to learn what components will be deployed.

## Step 2. Create your app

Let's start by creating a few files that compose our Python web app.  You can also find all these files in the `docs/flask` directory of the [examples](https://github.com/acorn-io/examples) GitHub repo if you'd rather start from there.

```shell
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

```docker title="acorn-test-app/Dockerfile"
FROM python:3-alpine
WORKDIR /app
ENV FLASK_APP=app.py
ENV FLASK_RUN_HOST=0.0.0.0
RUN apk add --no-cache gcc musl-dev linux-headers
ADD requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE 5000
CMD ["flask", "run"]
```

## Step 3. Author your Acornfile

Create your Acornfile with the following contents. Each item will be explained below, this file demonstrates a lot of what Acorn can do.

```acorn title="acorn-test-app/Acornfile"
args: {
  // Configure your personal welcome text
  welcome: "Hello Acorn User!!"
}

containers: {
  app: {
    build: "."
    env: {
      "PG_USER": "postgres"
      "PG_PASS": "secret://quickstart-pg-pass/token"
      "WELCOME": args.welcome
      if args.dev { "FLASK_ENV": "development" }
    }
    dependsOn: [
      "db",
      "cache"
    ]
    if args.dev { dirs: "/app": "./" }
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
        "/var/lib/postgresql/data": "volume://pgdata?subpath=data"
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

- `args` section: describes a set of arguments that can be passed in by the user of this Acorn image
  - A help text will be auto-generated using the comment just above the arg:

    ```bash
    $ acorn run . --help
    Usage of Acornfile:
          --welcome string   Configure your personal welcome text
    ```

- `containers` section: describes the set of containers your Acorn app consists of
  - Note: `app`, `db` and `cache` are custom names of your containers
  - `app` - Our Python Flask App
    - `build`: build from Dockerfile that we created
    - `env`: environment variables, statically defined, referencing a secret or referencing an Acorn argument
    - `dependsOn`: dependencies that have to be up and running before the app is started (here it is waiting for the database and the cache to be running)
    - `ports`: using the `publish` type, we expose the app inside the cluster but also outside of it using an auto-generated ingress resource ([more on this later](#step-5-access-your-app))
    - `dirs`: Directories to mount into the container filesystem
      - `if !args.dev`: The following block applies only if built-in development mode is **disabled**. ([more on the development mode later](#step-6-development-mode))
      - `dirs: "/app": "./"`: Mount the current directory to the /app dir, which is where the code resides inside the container as per the `Dockerfile`. This is to enable hot-reloading of code.
  - `cache` - Redis
    - `image`: existing OCI/Docker image to use (here: from DockerHub library)
    - `ports`: no type defined, defaults to `internal`, which makes it available to the other containers in this Acorn app
  - `db` - Postgres Database Server
    - `image`, `env`,`ports`: nothing new here
    - `dirs`: Directories to mount into the container filesystem
      - `if !args.dev`: The following block applies only if the built-in development mode is **disabled** ([more on the development mode later](#step-6-development-mode))
      - `volume://pgdata?subpath=data`: references a volume defined in the top-level `volumes` section in the Acornfile and specifies the subpath `data` as the mountpoint.
    - `files`: Similar to `dirs` but only for files. Additionally, content can be created in-line and even utilize generating functions.
- `localData`: Set of variables for this Acorn app
  - `food`: Custom variable, defining a list of food which is accessed in `containers.db.files` to pre-fill the database.
- `volumes`: (persistent) data volumes to be used by any container in the Acorn app
  - `pgdata` custom volume name, referenced in `containers.db.dirs`
    - `accessModes`: (list of) modes to allow access to this volume
- `secrets`: set of secrets that can be auto-generated and used by any container in the Acorn app
  - `quickstart-pg-pass`: custom secret name, referenced by `containers.app.env` and `containers.db.env`
    - `type`: There are several [secret types](38-authoring/05-secrets.md#types-of-secrets). Here, a token (random string) will be generated for you at runtime.

## Step 4. Run your Acorn app

To start your Acorn app just run:

```bash
acorn run -n awesome-acorn .
```

or customize the welcome text argument via:

```bash
acorn run -n awesome-acorn . --welcome "Let's Get Started"
```

The `-n awesome-acorn` gives this app a specific name so that the rest of the steps can refer to it.  If you omit `-n`, a random two-word name will be generated.

## Step 5. Access your app

Due to the configuration `ports: publish: "5000/http"` under `containers.app`, our web app will be exposed outside of our Kubernetes cluster using the cluster's ingress controller.
Checkout the running apps via

```bash
acorn apps
```

```bash
$ acorn apps
NAME         IMAGE          HEALTHY   UP-TO-DATE   CREATED    ENDPOINTS                                             MESSAGE
awesome-acorn   2d73c8a0493f   3         3            121m ago   http://app.awesome-acorn.local.on-acorn.io => app:5000   OK
```

You probably already noticed the link right there in the `ENDPOINTS` column. It will take you to your Python Flask App.

## Step 6. Development Mode

Now that we have a way to package and deploy our app, lets look at how we can configure the Acornfile to enable the development flow. In this mode, we will be able to make changes and see them updated inside the app container in real time.

To enable the Acorn development mode, first stop the app and then re-run with the `-i` flag.

```bash
acorn stop awesome-acorn
acorn run -n awesome-acorn -i .
```

In development mode, Acorn will watch the local directory for changes and synchronize them to the running Acorn app.
In general, changes to the Acornfile are directly synchronized, e.g. adding environment variables, etc.
Depending on the change, the deployed containers will be recreated.

The following lines additionally enable hot-reloading of code by mounting the current local directory into the app container:

```acorn
containers: {
  app: {
    // ...
    if args.dev { dirs: "/app": "./" }
    //...
  }
  //...
}
```

In this case, additionally `if args.dev { "FLASK_ENV": "development" }` enables Flask's development mode.

Running in development mode, Acorn will keep a session open, streaming all the container logs to your terminal and notifying you of any changes that are happening.  Press `Ctrl-c` to end the session and terminate the running app.

To test it, you can change something in the `app.py`.
For example, add a line to the HTML template at the top and change it to

```python
# HTML Jinja2 Template which will be shown in the browser
page_template = '''
        <div style="margin: auto; text-align: center;">
        <h1>{{ welcome_text }}</h1><br>
        <h2>This is a change :)</h2>
        You're visitor #{{ visitors }} to learn what squirrels love the most:<br>
        <ul>
            {%- for food in foods %}
            <li>{{ food }}</li>
            {%- endfor %}
        </ul>
        </div>
        '''
```

You will see the change applied when when you reload the application's page in your browser.

## Step 7. Build and Push your Acorn image

Ready to release your Acorn app into the wild?
Let's package it up in a single Acorn image and distribute it via an OCI registry.

> **Note**: This example uses GitHub's container registry `ghcr.io`.
> You want to push to DockerHub instead? The prefix is `docker.io`.
> Just make sure that you actually have write access to the target repository.

```bash
# Login into your OCI registry, if needed (interactive)
acorn login ghcr.io

# Build the Acorn image and tag it to your liking
acorn build -t ghcr.io/acorn-io/getting-started:v0.0.1 .

# Push the newly built Acorn image
acorn push ghcr.io/acorn-io/getting-started:v0.0.1
```

Now, everyone else can run your Acorn image via

```bash
acorn run --name awesome-acorn ghcr.io/acorn-io/getting-started:v0.0.1
```

## Interacting with an Acorn app

### Execute a command inside the running container

You can get an interactive shell into any running app or container with:

```bash
acorn exec awesome-acorn
```

If there is more than one container in an app, you will be prompted to pick one. You can also run a specific command instead of getting a shell:

```bash
acorn exec awesome-acorn env
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

### Start and Stop your app

That's easy!

```shell
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

> Dangerous Pro-Tip: to remove **all** Acorn apps at once: `acorn rm $(acorn apps -qa)`

## What's next?

- [Explore all the other awesome Acorn commands](100-reference/01-command-line/acorn.md)
- [Read through the Acornfile reference](100-reference/03-acornfile.md)
- [Have a look what makes up Acorn](60-architecture/01-ten-thousand-foot-view.md)
- [Try some of our other example Acorns](https://github.com/acorn-io/examples)
