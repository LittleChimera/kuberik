# kuberik

<img src="./docs/.vuepress/public/assets/img/logo.svg" height=100 />

----

Kuberik is an extensible pipeline engine for Kubernetes. It enables
execution of pipelines on top of Kubernetes by leveraging full expressiveness
of Kubernetes API.

----

## Project status

**Project status:** *alpha*

Project is in alpha stage. API is still a subject to change. Please do not use it in production.

## Development

### Requirements
  - Go 1.13
  - Operator SDK 0.15.1

### Prerequisites
  - authenticated to a Kubernetes cluster (e.g. [Kubernetes from Docker on Desktop](https://docs.docker.com/docker-for-mac/kubernetes/))
  - applied kuberik CRDs on the cluster
    - `kubectl apply -f deploy/crds`

Start up the operator:

```shell
operator-sdk run --local --namespace=<your-namespace>
```

You can use one of the pipelines from the `docs/examples` directory to execute some workload on kuberik.
```shell
kubectl apply -f docs/examples/hello-world.yaml
```

Install kuberik CMD:

```shell
go install ./cmd/kuberik
```

Trigger the pipeline with `kuberik` cmd.
```shell
kuberik create play --from=hello-world
```
