# github-status-sync

> Tekton Interceptor to sync Tekton status with GitHub based on [Cloud Event](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents)

## Overview

Tekton offers has [a couple of services](https://github.com/tektoncd/experimental) which help to sync Tekton `TaskRun`
or `PipelineRun` statuses with GitHub. Those services are great, but they are based on
a [Reconciler](https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/reconciler)
pattern that makes them slow to process a big queue, and synchronous.

`github-status-sync` helps to sync Tekton statuses with GitHub by using
an [Interceptor](https://tekton.dev/vault/triggers-main/clusterinterceptors/) which accepts
[Cloud Event](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents). This approach helps to achieve **near
real-time synchronization**.

## Service Configuration

`github-status-sync` can be configured by using environment variables, a configuration file, or flags.

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

By default, `github-status-sync` lookups a configuration file in the following order:

1. `$HOME/.config/github-status-sync/config.yaml`
2. `/etc/config/github-status-sync/config.yaml`
3. `$PWD/config/github-status-sync/config.yaml`

Also, `github-status-sync` allows to set a path to a configuration file by using a `--config` flag:

```shell
github-status-sync --config=$PWD/github-status-sync.yaml
```

### Flags

| Flag Name                | Description                                                                                                                                                                      | Required | Default |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `config`                 | The path to the config file.                                                                                                                                                     | No       | `""`    |
| `github-app-id`          | GitHub App ID. Can be found at <https://github.com/settings/apps> under `Edit > General > About > App ID`                                                                        | Yes      | `""`    |
| `github-installation-id` | GitHub [Installation ID](https://docs.github.com/en/enterprise-server@2.20/developers/webhooks-and-events/webhook-events-and-payloads#webhook-payload-object-common-properties). | Yes      | `""`    |
| `github-app-key`         | GitHub [App Private Key](https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#generating-a-private-key).                              | Yes      | `""`    |

## Interceptor Configuration

`github-status-sync` uses annotations with the prefix `github.tekton.dev` to identify and track `TaskRun` published
via [Cloud Event](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents).

| Annotation Name           | Description                                                                                                                                                                                                                                                                                                                                            |
|---------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `github.tekton.dev/owner` | GitHub org or user who owns the repo (for `ElementalCognition/tekton-toolbox`, this should be `ElementalCognition`).                                                                                                                                                                                                                                   |
| `github.tekton.dev/repo`  | GitHub repo name (for `ElementalCognition/tekton-toolbox`, this should be `tekton-toolbox`).                                                                                                                                                                                                                                                           |
| `github.tekton.dev/ref`   | GitHub Git Ref.                                                                                                                                                                                                                                                                                                                                        |
| `github.tekton.dev/url`   | Details URL to use for GitHub CheckRun/Status. If not specified, defaults to `https://tekton.dev/#/namespaces/{{ .Namespace }}/taskruns/{{ .Name }}`. You can use `text/template` templating syntax to generate URL and access any variables of [`TaskRun`](https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1#TaskRun) inside. |
| `github.tekton.dev/name`  | Display name to use for GitHub CheckRun/Status. If not specified, defaults to `{{ .Namespace }}/{{ .Name }}`. You can use `text/template` templating syntax to generate name and access any variables of [`TaskRun`](https://github.com/tektoncd/pipeline/blob/main/pkg/apis/pipeline/v1beta1/taskrun_types.go) inside.                                |

Sample `TaskRun` file:

```yaml
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  annotations:
    github.tekton.dev/owner: ElementalCognition
    github.tekton.dev/repo: tekton-toolbox
    github.tekton.dev/ref: deadbeef
    github.tekton.dev/url: >-
      https://tekton.dev/#/namespaces/{{ .Namespace }}/pipelineruns/{{ index .Labels "tekton.dev/pipelineRun" }}?pipelineTask={{ index .Labels "tekton.dev/pipelineTask" }}
    github.tekton.dev/name: >-
      {{ index .Labels "tekton.dev/pipeline" }} / {{ index .Labels "tekton.dev/pipelineTask" }}
```
