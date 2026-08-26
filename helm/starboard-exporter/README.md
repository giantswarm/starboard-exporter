# starboard-exporter

A Helm chart for starboard-exporter, which exposes Prometheus metrics from Aqua VulnerabilityReport and other custom resources.

**Homepage:** <https://github.com/giantswarm/starboard-exporter>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicas | int | `1` |  |
| registry.domain | string | `"gsoci.azurecr.io"` |  |
| image.name | string | `"giantswarm/starboard-exporter"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| global.podSecurityStandards.enforced | bool | `true` |  |
| pod.user.id | int | `1000` |  |
| pod.group.id | int | `1000` |  |
| nodeSelector | object | `{}` |  |
| tolerations | list | `[]` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.privileged | bool | `false` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| resources.requests.cpu | string | `"100m"` |  |
| resources.requests.memory | string | `"220Mi"` |  |
| resources.limits.cpu | string | `"100m"` |  |
| resources.limits.memory | string | `"220Mi"` |  |
| autoMemoryLimit.enabled | bool | `true` |  |
| autoMemoryLimit.ratio | float | `0.9` |  |
| exporter.requeueMaxJitterPercent | int | `10` |  |
| exporter.configAuditReports.enabled | bool | `true` |  |
| exporter.vulnerabilityReports.targetLabels | list | `[]` |  |
| exporter.vulnerabilityReports.scanners.trivy.enabled | bool | `true` |  |
| exporter.vulnerabilityReports.scanners.kubescape.enabled | bool | `false` |  |
| monitoring.serviceMonitor.enabled | bool | `true` |  |
| monitoring.serviceMonitor.labels | object | `{}` |  |
| monitoring.serviceMonitor.relabelings[0].action | string | `"labeldrop"` |  |
| monitoring.serviceMonitor.relabelings[0].regex | string | `"pod|service|container"` |  |
| monitoring.serviceMonitor.metricRelabelings | list | `[]` |  |
| monitoring.grafanaDashboard.enabled | bool | `true` |  |
| networkpolicy.enabled | bool | `true` |  |
| podAnnotations | object | `{}` |  |
| minReplicas | int | `2` |  |
| maxReplicas | int | `97` |  |
| customMetricsHPA.enabled | bool | `false` |  |
| customMetricsHPA.minReplicas | int | `2` |  |
| customMetricsHPA.maxReplicas | int | `97` |  |
| customMetricsHPA.metricName | string | `"scrapedurationseconds"` |  |
| customMetricsHPA.targetAverageValueSeconds | int | `10` |  |
| verticalPodAutoscaler.enabled | bool | `true` |  |
| verticalPodAutoscaler.containerPolicies.minAllowed.cpu | string | `"50m"` |  |
| verticalPodAutoscaler.containerPolicies.minAllowed.memory | string | `"100Mi"` |  |
| verticalPodAutoscaler.containerPolicies.maxAllowed.cpu | int | `1` |  |
| verticalPodAutoscaler.containerPolicies.maxAllowed.memory | string | `"4Gi"` |  |
| kedaScaledObject.enabled | bool | `false` |  |
| kedaScaledObject.minReplicas | int | `2` |  |
| kedaScaledObject.maxReplicas | int | `97` |  |
| kedaScaledObject.triggers | list | `[]` |  |
