# GitHub Pipeline Config

> Tekton Interceptor for retrieving [`.tekton.yaml`](./pipeline-config.md) from GitHub

## Overview

`github-pipeline-config` loads [`pipeline-config`](./pipeline-config.md) from GitHub, attempts to read an existing configuration from [`InterceptorRequest#extensions["pipeline-config"]`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest), merges them together, and returns merged result as [`InterceptorResponse#extensions["pipeline-config"]`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorResponse).

## Service Configuration

`github-pipeline-config` can be configured using environment variables, a configuration file, or flags.

### Environment Variables

| Environment Variable     | Description                                                                                                                                                                      | Required | Default        |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------|
| `ADDR`                   | The address and port.                                                                                                                                                            | No       | `"0.0.0.0:80"` |
| `GITHUB_APP_ID`          | GitHub App ID. Can be found at <https://github.com/settings/apps> under `Edit > General > About > App ID`                                                                        | Yes      | `""`           |
| `GITHUB_INSTALLATION_ID` | GitHub [Installation ID](https://docs.github.com/en/enterprise-server@2.20/developers/webhooks-and-events/webhook-events-and-payloads#webhook-payload-object-common-properties). | Yes      | `""`           |
| `GITHUB_APP_KEY`         | GitHub [App Private Key](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#generating-a-private-key).                              | Yes      | `""`           |

### Configuration File

| Field Name               | Description                                                                                                                                                                      | Required | Default        |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|----------------|
| `addr`                   | The address and port.                                                                                                                                                            | No       | `"0.0.0.0:80"` |
| `github-app-id`          | GitHub App ID. Can be found at <https://github.com/settings/apps> under `Edit > General > About > App ID`                                                                        | Yes      | `""`           |
| `github-installation-id` | GitHub [Installation ID](https://docs.github.com/en/enterprise-server@2.20/developers/webhooks-and-events/webhook-events-and-payloads#webhook-payload-object-common-properties). | Yes      | `""`           |
| `github-app-key`         | GitHub [App Private Key](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#generating-a-private-key).                              | Yes      | `""`           |

Sample configuration file:

```yaml
github-app-id: "12345"
github-installation-id: "6789"
github-app-key: "/etc/config/github/privateKey.pem"
```

By default, `github-pipeline-config` searches for a configuration file in the following order:

1. `$HOME/.config/github-pipeline-config/config.yaml`
2. `/etc/config/github-pipeline-config/config.yaml`
3. `$PWD/config/github-pipeline-config/config.yaml`

Additionally, `github-pipeline-config` allows setting a path to a configuration file using the `--config` flag:

```shell
github-pipeline-config --config=$PWD/github-pipeline-config.yaml
```

### Flags

| Flag Name                | Description                                                                                                                                                                      | Required | Default |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `config`                 | The path to the config file.                                                                                                                                                     | No       | `""`    |
| `github-app-id`          | GitHub App ID. Can be found at <https://github.com/settings/apps> under `Edit > General > About > App ID`                                                                        | Yes      | `""`    |
| `github-installation-id` | GitHub [Installation ID](https://docs.github.com/en/enterprise-server@2.20/developers/webhooks-and-events/webhook-events-and-payloads#webhook-payload-object-common-properties). | Yes      | `""`    |
| `github-app-key`         | GitHub [App Private Key](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#generating-a-private-key).                              | Yes      | `""`    |

## Interceptor Configuration

`github-pipeline-config` allows to specify `namespace` and `name` parameters to read Kubernetes ConfigMap.

| Parameter Name | Description                                                                                                          |
|----------------|----------------------------------------------------------------------------------------------------------------------|
| `owner`        | GitHub org or user who owns the repo (for `ElementalCognition/tekton-toolbox`, this should be `ElementalCognition`). |
| `repo`         | GitHub repo name (for `ElementalCognition/tekton-toolbox`, this should be `tekton-toolbox`).                         |
| `ref`          | GitHub Git Ref.                                                                                                      |

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
        - name: owner
          value: body.repository.owner.login
        - name: repo
          value: body.repository.name
        - name: ref
          value: >-
            "pull_request" in body
              ? body.pull_request.head.sha
              : body.head_commit.id
      ref:
        kind: ClusterInterceptor
        name: github-pipeline-config
```
