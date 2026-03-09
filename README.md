# vismo

The vismo tool is really helpful when you want to know where your live Kubernetes resources come from. You do not have to spend a lot of time trying to figure out what is going on. Instead you can use that time to fix things.

---

## What vismo does

- Vismo fetches any resource in the way that kubectl does. You can use the names and the same slash syntax. You can also use the -n flag.

- Vismo detects who manages each resource. This could be Helm or ArgoCD or Kustomize or Flux or plain kubectl. Sometimes it is a mix of these.

- Vismo traces field ownership. For example if you want to know who last changed the spec.replicas field vismo can tell you. It does this by looking at the Kubernetes managedFields.

---

## Installation

```bash
git clone https://github.com/Veinar/vismo
cd vismo
make build # this creates the bin/vismo file
```

To use vismo you need to have Go 1.25 or later installed. You also need to be able to reach a Kubernetes cluster. You can do this with a kubeconfig file or by running vismo inside the cluster.

---

## Usage

```bash
# Fetch a deployment and see who manages it

vismo get deploy/api -n production

# You can also use the form, which works like kubectl

vismo get deployment/api -n production

# If you want to know who last set the spec.replicas field and when

vismo get deploy/api -n production --field spec.replicas

# List all deployments in a namespace

vismo get deployments -n production

# Print the version of vismo

vismo version
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--namespace` | `-n` | - | The Kubernetes namespace to use |
| `--all-namespaces` | `-A` | `false` | Whether to query across all namespaces |
| `--field` | - | - | The field path to trace ownership, like `spec.replicas` |
| `--kubeconfig` | - | `~/.kube/config` | The path to your kubeconfig file |
| `--log-level` | - | `warn` | How much log output you want: `debug`, `info`, `warn`, or `error` |

---

## Example output

```

--- Resource YAML ---

apiVersion: apps/v1

kind: Deployment

...

--- Detected Sources ---

1. Type: Helm

Name: api

chart: api-2.1.0

--- Field Ownership Trace ---

Field: spec.replicas

Final Value: 5

Ownership Trace (based on managedFields):

1. Manager: 'kubectl-scale'

Operation: Update

Timestamp: 2023-10-27T10:00:05Z

(This is the final owner of the value)

2. Manager: 'helm'

Operation: Update

Timestamp: 2023-10-27T09:30:12Z

```

Here is what happened: Helm originally set the replicas field, but then `kubectl scale` ran later and overwrote it to 5. That is the final value, and `kubectl-scale` is the final owner. vismo read all of this from `managedFields` so you did not have to.

---

## Debugging

If you get stuck you can run vismo with `--log-level=debug` to see every step it takes. This includes which API calls it makes, how the GVR is resolved, and which `managedFields` entries are scanned.

```bash

vismo get deploy/api -n production --log-level=debug

```

---

## Supported resource aliases

Vismo works with all the short names. These include deploy, sts, svc, po, cm, secret, ing, ds, rs, cj, job, pvc, pv, ns, sa and no. Also accepting plurals.