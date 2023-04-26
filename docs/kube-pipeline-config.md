# kube-pipeline-config

> Tekton Interceptor to get [`pipeline-config`](./pipeline-config.md) from Kubernetes ConfigMap

## Overview

`kube-pipeline-config` reads [`pipeline-config`](./pipeline-config.md) from Kubernetes ConfigMap, then tries to read
existing one
from [`InterceptorRequest#extensions["pipeline-config"]`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest)
, merges them together, and returns merged result
as [`InterceptorResponse#extensions["pipeline-config"]`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorResponse)
.

## Service Configuration

`kube-pipeline-config` can be configured by using environment variables, a configuration file, or flags.

### Environment Variables

| Environment Variable | Description           | Required | Default        |
|----------------------|-----------------------|----------|----------------|
| `ADDR`               | The address and port. | No       | `"0.0.0.0:80"` |

### Configuration File

| Field Name | Description           | Required | Default        |
|------------|-----------------------|----------|----------------|
| `addr`     | The address and port. | No       | `"0.0.0.0:80"` |

Sample configuration file:

```yaml
addr: "0.0.0.0:80"
```

By default, `kube-pipeline-config` lookups a configuration file in the following order:

1. `$HOME/.config/kube-pipeline-config/config.yaml`
2. `/etc/config/kube-pipeline-config/config.yaml`
3. `$PWD/config/kube-pipeline-config/config.yaml`

Also, `kube-pipeline-config` allows to set a path to a configuration file by using a `--config` flag:

```shell
kube-pipeline-config --config=$PWD/kube-pipeline-config.yaml
```

### Flags

| Flag Name | Description                  | Required | Default |
|-----------|------------------------------|----------|---------|
| `config`  | The path to the config file. | No       | `""`    |

## Interceptor Configuration

`kube-pipeline-config` allows to specify `namespace` and `name` parameters to read Kubernetes ConfigMap.

| Parameter Name | Description                                         |
|----------------|-----------------------------------------------------|
| `namespace`    | The namespace where to lookup Kubernetes ConfigMap. |
| `name`         | The name of Kubernetes ConfigMap.                   |

Sample `Trigger` file:

```yaml
---
apiVersion: triggers.tekton.dev/v1beta1
kind: Trigger
metadata:
  name: my-trigger
spec:
  interceptors:
    - params:
        - name: name
          value: my-pipeline-config-cm
        - name: namespace
          value: tekton
      ref:
        kind: ClusterInterceptor
        name: kube-pipeline-config
```

`kube-pipeline-config` **watches for any further changes** of received Kubernetes ConfigMap, but it should be properly
annotated with `labels.name` equals to `pipeline-config-cm` and has `config.yaml` field, eg. `my-pipeline-config-cm`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-pipeline-config-cm
  namespace: tekton
  labels:
    name: pipeline-config-cm
data:
  config.yaml: |
```
