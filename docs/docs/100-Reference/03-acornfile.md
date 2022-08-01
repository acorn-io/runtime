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

```cue
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

```cue
containers: web: {
	image: "nginx"
	ports: publish: "80/http"
}
```

### dirs, directories
### files
### image
### build
### command, cmd
### interactive, tty, stdin
### entrypoint
### environment, env
### workingDir, workDir
### dependsOn
### ports
### probes, probe
### scale

## sidecars
### init

## jobs
### schedule

## volumes
### size
### accessModes, accessMode

## secrets
### type
### params
### data

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
