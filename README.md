# Tekton Toolbox

> A toolbox for Tekton

## Overview

A collection of tools and services designed to streamline working with Tekton:

- [`gcs-log-proxy`](./docs/gcs-log-proxy.md) - A proxy for loading Tekton external logs from Google Cloud Storage.
- [`github-pipeline-config`](./docs/github-pipeline-config.md) - Tekton Interceptor for retrieving [`pipeline-config`](./docs/pipeline-config.md) from GitHub.
- [`github-status-sync`](./docs/github-status-sync.md) - Tekton Interceptor for synchronizing Tekton status with GitHub using [Cloud Events](https://tekton.dev/docs/pipelines/events/#events-via-cloudevents).
- [`kube-pipeline-config`](./docs/kube-pipeline-config.md) - Tekton Interceptor for obtaining [`pipeline-config`](./docs/pipeline-config.md) from Kubernetes ConfigMap.
- [`pipeline-config-trigger`](./docs/pipeline-config-trigger.md) - Tekton Interceptor for retrieving a list of Tekton `PipelineRun` from [`pipeline-config`](./docs/pipeline-config.md) and triggering them.

## Interceptor Setup

To deploy an interceptor (the same approach works for each cluster interceptor listed above), e.g., `kube-pipeline-config`:

The k8s deployment must include the following environment variables:
```yaml
- name: SYSTEM_NAMESPACE
  value: tekton-pipelines
- name: INTERCEPTER_NAME  # Keep the k8s service name and clusterinterceptor name the same.
  value: kube-pipeline-config
- name: SVC_NAMESPACE
  valueFrom:
    fieldRef:
      fieldPath: metadata.namespace
```

The k8s deployment must use HTTPS for readiness and liveness checks:
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

The interceptor will only listen on port 8443 for HTTPS connections. When starting, the interceptor checks if a secret (with the same name as the interceptor) containing certificates exists. If it's missing, the interceptor will create one and populate it with the necessary data. On subsequent starts and/or redeployments, it will check for the existence of the secret and use the existing certificates. A custom resource `kind: ClusterInterceptor` will be created by the interceptor and updated with the `caBundle` using the `ca-cert.pem` from the secret.

> IMPORTANT

If the certificate secret is deleted, certificates will be regenerated, and the `caBundle` will be updated accordingly. You **MUST** restart `deploy/el-github-listener` and `deploy/el-events-listener`, otherwise the events-listeners will encounter an `X509 SelfSign certificate` error.
