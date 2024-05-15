# pipeline-config

## Overview

[`Config`](../pkg/pipelineconfig/config.go) allows to specify default values and triggers to create a list
of [`PipelineRun`](https://tekton.dev/docs/pipelines/pipelineruns/) based
on [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest).

## Specifying Config

[`Config`](../pkg/pipelineconfig/config.go) supports supports the following fields:

- `defaults` - Specifies common default values for each [`Pipeline`](#specifying-defaults).
- `triggers` - Specifies a list of [`Trigger`](#specifying-triggers).

Sample config file:

```yaml
defaults:
  timeouts:
    pipeline: 1h
    tasks: 30m
    finally: 10m
  metadata:
    namespace: tekton
    annotations:
      github.tekton.dev/owner: body.repository.owner.login
      github.tekton.dev/repo: body.repository.name
      github.tekton.dev/commit: >-
        "pull_request" in body
          ? body.pull_request.head.sha
          : body.head_commit.id
      github.tekton.dev/url: >-
        https://dashboard.tekton.dev/#/namespaces/{{ .Namespace }}/pipelineruns/{{ index .Labels "tekton.dev/pipelineRun" }}?pipelineTask={{ index .Labels "tekton.dev/pipelineTask" }}
      github.tekton.dev/name: >-
        {{ index .Labels "tekton.dev/pipeline" }} / {{ index .Labels "tekton.dev/pipelineTask" }}
  params:
    - name: clone-url
      value: body.repository.clone_url
    - name: clone-ref
      value: >-
        "pull_request" in body
          ? body.pull_request.head.sha
          : body.head_commit.id
    - name: repo-name
      value: body.repository.name
    - name: repo-owner-name
      value: body.repository.owner.login
  workspaces:
    - name: source
      volumeClaimTemplate:
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 5Gi
          storageClassName: <Storage Class Name>
    - name: cache
      persistentVolumeClaim:
        claimName: <PVC name>
  podTemplate:
    tolerations:
      - effect: NoSchedule
        key: tekton-pipelines
        operator: Equal
        value: "true"
    nodeSelector:
      node_pool: tekton-pipelines
triggers:
  - name: pr
    filter: >-
      "action" in body &&  body.action in ["opened", "synchronize", "reopened"]
    pipelines: [ ]
    defaults:
      params: [ ]
  - name: main
    filter: >-
      "ref" in body && body.ref == "refs/heads/main"
    pipelines: [ ]
    defaults:
      params: [ ]
  - name: tag
    filter: >-
      "ref" in body && body.ref.startsWith("refs/tags/")
    defaults:
      params: [ ]
```

## Specifying Defaults

`Defaults` definition contains all [`Pipeline`](#specifying-pipeline) fields except `name`.

## Specifying Triggers

Each [`Triggers`](../pkg/pipelineconfig/trigger.go) definition supports the following fields:

- `name` - Specifies a name of current trigger.
- `filter` - Specifies [CEL expression](https://github.com/google/cel-spec) which helps
  to [filter](../pkg/pipelineconfig/trigger_filter.go) out triggers based
  on [`InterceptorRequest`](https://pkg.go.dev/github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1#InterceptorRequest)
  .
- `pipelines` - Specifies a list of [`Pipeline`](#specifying-pipeline).
- `defaults` - Specifies default values for each [`Pipeline`](#specifying-pipeline).

## Specifying Pipeline

`Pipeline` definition supports the following fields:

- `*` -
  all [`PipelineRunSpec`](https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/apis/pipeline/v1#PipelineRunSpec)
  fields.
- `name` - Specifies a name of current pipeline.
- `metadata` - Specifies [`metadata`](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#ObjectMeta) that uniquely
  identifies pipeline, eg. `annotations`.


## Example on how to rewrite computeResources for task or task.step
``` .tekton.yaml
triggers:
  - name: pr
    pipelines:
      - name: buf
        pipelineRef:
          resolver: git
          params:
            - name: repo
              value: tekton-catalog-new
            - name: pathInRepo
              value: pipeline/protos/0.3/buf.yaml
        taskRunSpecs:
          - pipelineTaskName: lint
            computeResources:
              requests:
                memory: 1Gi
                cpu: '1'
              limits:
                memory: 1Gi
                cpu: '1'

          - pipelineTaskName: svu
            stepSpecs:
              - name: svu
                computeResources:
                  requests:
                    memory: 1.2Gi
                    cpu: '1.2'
                  limits:
                    memory: 1.2Gi
                    cpu: '1.2'
```
