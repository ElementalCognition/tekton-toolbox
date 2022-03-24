# gcs-log-proxy

> A proxy to load Tekton external logs from Google Cloud Storage

## Overview

Tekton offers a way
to [load logs from an external service](https://github.com/tektoncd/dashboard/blob/main/docs/walkthrough/walkthrough-logs.md)
if Tekton can't find them due to garbage collection by the cluster.

`gcs-log-proxy` helps to load external logs from Google Cloud Storage.

In addition, [`tekton-toolbox`](../README.md) has [`pkg/logproxy`](../pkg/logproxy) package which helps to implement a
new services to poxy logs by writing less boilerplate code. For example, please see [`pkg/logproxy`](../pkg/logproxy)
, [`pkg/gcslogproxy`](../pkg/gcslogproxy), and [`cmd/gcs-log-proxy`](../cmd/gcs-log-proxy).

## Service Configuration

`gcs-log-proxy` can be configured by using environment variables, a configuration file, or flags.

### Environment Variables

| Environment Variable | Description                          | Required | Default            |
|----------------------|--------------------------------------|----------|--------------------|
| `BUCKET`             | The logs bucket.                     | Yes      | `""`               |
| `ADDR`               | The address and port.                | No       | `"0.0.0.0:80"`     |
| `WORKERS`            | The number of workers to fetch logs. | No       | `runtime.NumCPU()` |

### Configuration File

| Field Name | Description                          | Required | Default            |
|------------|--------------------------------------|----------|--------------------|
| `bucket`   | The logs bucket.                     | Yes      | `""`               |
| `addr`     | The address and port.                | No       | `"0.0.0.0:80"`     |
| `workers`  | The number of workers to fetch logs. | No       | `runtime.NumCPU()` |

Sample configuration file:

```yaml
bucket: "tekton-logs"
addr: "0.0.0.0:80"
workers: 8
```

By default, `gcs-log-proxy` lookups a configuration file in the following order:

1. `$HOME/.config/gcs-log-proxy/config.yaml`
2. `/etc/config/gcs-log-proxy/config.yaml`
3. `$PWD/config/gcs-log-proxy/config.yaml`

Also, `gcs-log-proxy` allows to set a path to a configuration file by using a `--config` flag:

```shell
gcs-log-proxy --config=$PWD/gcs-log-proxy.yaml
```

### Flags

| Flag Name | Description                          | Required | Default            |
|-----------|--------------------------------------|----------|--------------------|
| `config`  | The path to the config file.         | No       | `""`               |
| `bucket`  | The logs bucket.                     | Yes      | `""`               |
| `addr`    | The address and port.                | No       | `"0.0.0.0:80"`     |
| `workers` | The number of workers to fetch logs. | No       | `runtime.NumCPU()` |
