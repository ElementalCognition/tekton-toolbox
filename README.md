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

## Interceptors setup

To deploy interceptor (the same approach works for each cluster interceptor listed abouve) e.g `kube-pipeline-config`

k8s deployment must have the following ENV vars:
```yaml
- name: SYSTEM_NAMESPACE
  value: tekton-pipelines
- name: INTERCEPTER_NAME  # Keep k8s service name and clusterintercepter name the same.
  value: kube-pipeline-config
- name: SVC_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

k8s deployment must use https to check readiness and liveness:
```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8443
    scheme: HTTPS
  initialDelaySeconds: 5
livenessProbe:
  httpGet:
    path: /health
    port: 8443
    scheme: HTTPS
```

Interceptor will listens only on 8443 port HTTPS on start, an interceptor will check if the secret (the secret name has the interceptor's name) with certificates exists.
If it's missing interceptor will create one and fill it with the data. Next start and/or redeploy will check if it exists and use existing certs.
Custom resource `kind: ClusterInterceptor` will be created by the interceptor and updated with `caBundle` taking `ca-cert.pem` from the secret.

> IMPORTANT

If a cert secret was deleted, certificates will be regenerated and `caBundle` will be updated accordingly. You <span style="color:red">**MUST**</span> restart `deploy/el-github-listener` and `deploy/el-events-listener` otherwise events-listeners will with `X509 SelfSign certificate` error.
