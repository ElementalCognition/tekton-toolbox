# pipeline-config-trigger

> Tekton Interceptor to get a list of Tekton `PipelineRun` from [`pipeline-config`](./pipeline-config.md) and trigger them

## Overview

`pipeline-config-trigger` reads [`pipeline-config`](./pipeline-config.md)
from [`InterceptorRequest#extensions["pipeline-config"]`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest)
, builds a list of `PipelineRun` based
on [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest)
, and triggers them after asynchronously.

## Service Configuration

`pipeline-config-trigger` can be configured by using environment variables, a configuration file, or flags.

### Environment Variables

| Environment Variable | Description                                    | Required | Default            |
|----------------------|------------------------------------------------|----------|--------------------|
| `ADDR`               | The address and port.                          | No       | `"0.0.0.0:80"`     |
| `WORKERS`            | The number of workers to create `PipelineRun`. | No       | `runtime.NumCPU()` |

### Configuration File

| Field Name | Description                                    | Required | Default            |
|------------|------------------------------------------------|----------|--------------------|
| `addr`     | The address and port.                          | No       | `"0.0.0.0:80"`     |
| `workers`  | The number of workers to create `PipelineRun`. | No       | `runtime.NumCPU()` |

Sample configuration file:

```yaml
addr: "0.0.0.0:80"
workers: 8
```

By default, `pipeline-config-trigger` lookups a configuration file in the following order:

1. `$HOME/.config/pipeline-config-trigger/config.yaml`
2. `/etc/config/pipeline-config-trigger/config.yaml`
3. `$PWD/config/pipeline-config-trigger/config.yaml`

Also, `pipeline-config-trigger` allows to set a path to a configuration file by using a `--config` flag:

```shell
pipeline-config-trigger --config=$PWD/pipeline-config-trigger.yaml
```

### Flags

| Flag Name | Description                                    | Required | Default            |
|-----------|------------------------------------------------|----------|--------------------|
| `config`  | The path to the config file.                   | No       | `""`               |
| `addr`    | The address and port.                          | No       | `"0.0.0.0:80"`     |
| `workers` | The number of workers to create `PipelineRun`. | No       | `runtime.NumCPU()` |
