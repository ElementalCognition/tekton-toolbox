# GCS Log Proxy

A proxy for loading external Tekton logs from Google Cloud Storage.

## Overview

Tekton provides a way to [load logs from an external service](https://github.com/tektoncd/dashboard/blob/main/docs/walkthrough/walkthrough-logs.md) when logs cannot be found due to garbage collection by the cluster.

`gcs-log-proxy` assists in loading external logs from Google Cloud Storage.

Furthermore, [`tekton-toolbox`](../README.md) contains the [`pkg/logproxy`](../pkg/logproxy) package that simplifies the implementation of new services for proxying logs by reducing the amount of boilerplate code. For examples, please refer to [`pkg/logproxy`](../pkg/logproxy), [`pkg/gcslogproxy`](../pkg/gcslogproxy), and [`cmd/gcs-log-proxy`](../cmd/gcs-log-proxy).

## Service Configuration

`gcs-log-proxy` can be configured using environment variables, a configuration file, or flags.

### Environment Variables

| Environment Variable | Description                          | Required | Default            |
|----------------------|--------------------------------------|----------|--------------------|
| `BUCKET`             | The log storage bucket.              | Yes      | `""`               |
| `ADDR`               | The address and port.                | No       | `"0.0.0.0:80"`     |
| `WORKERS`            | The number of workers for log fetch. | No       | `runtime.NumCPU()` |

### Configuration File

| Field Name | Description                          | Required | Default            |
|------------|--------------------------------------|----------|--------------------|
| `bucket`   | The log storage bucket.              | Yes      | `""`               |
| `addr`     | The address and port.                | No       | `"0.0.0.0:80"`     |
| `workers`  | The number of workers for log fetch. | No       | `runtime.NumCPU()` |

Sample configuration file:

```yaml
bucket: "tekton-logs"
addr: "0.0.0.0:80"
workers: 8
```

By default, `gcs-log-proxy` searches for a configuration file in the following order:

1. `$HOME/.config/gcs-log-proxy/config.yaml`
2. `/etc/config/gcs-log-proxy/config.yaml`
3. `$PWD/config/gcs-log-proxy/config.yaml`

Additionally, `gcs-log-proxy` allows setting a path to a configuration file using the `--config` flag:

```shell
gcs-log-proxy --config=$PWD/gcs-log-proxy.yaml
```

### Flags

| Flag Name | Description                          | Required | Default            |
|-----------|--------------------------------------|----------|--------------------|
| `config`  | The path to the config file.         | No       | `""`               |
| `bucket`  | The log storage bucket.              | Yes      | `""`               |
| `addr`    | The address and port.                | No       | `"0.0.0.0:80"`     |
| `workers` | The number of workers for log fetch. | No       | `runtime.NumCPU()` |
