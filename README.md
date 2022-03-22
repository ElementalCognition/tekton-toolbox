# tekton-toolbox

> Toolbox for Tekton

## Overview

A set of tools and services which simplify a process to work with Tekton.

- [`gcs-log-proxy`](./docs/gcs-log-proxy.md) - A proxy to load Tekton external logs from Google Cloud Storage.
- [`github-pipeline-config`](./docs/github-pipeline-config.md) - Tekton Interceptor to
  get [`pipeline-config`](./docs/pipeline-config.md) from GitHub.
- [`github-status-sync`](./docs/github-status-sync.md) - Tekton Interceptor to sync Tekton status with GitHub based
  on [Cloud Event](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents).
- [`kube-pipeline-config`](./docs/kube-pipeline-config.md) - Tekton Interceptor to
  get [`pipeline-config`](./docs/pipeline-config.md) from Kubernetes ConfigMap.
- [`pipeline-config-trigger`](./docs/pipeline-config-trigger.md) - Tekton Interceptor to get a list of
  Tekton `PipelineRun`
  from [`pipeline-config`](./docs/pipeline-config.md) and trigger them.
