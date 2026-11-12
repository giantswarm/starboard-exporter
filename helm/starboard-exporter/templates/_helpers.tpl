{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Render a string as a valid Kubernetes label value. Dev builds can produce
version strings that exceed the 63-character label limit or contain
characters that are invalid in a label value (e.g. the "+" that precedes
SemVer build metadata, or a "." left at the truncation point). Replace the
"+", truncate to 63 characters, and strip any trailing non-alphanumeric
characters so the value always ends with an alphanumeric character.
*/}}
{{- define "labels.sanitizeValue" -}}
{{- $value := . | toString | replace "+" "_" | trunc 63 -}}
{{- regexReplaceAll "[^A-Za-z0-9]+$" $value "" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "chart" -}}
{{- include "labels.sanitizeValue" (printf "%s-%s" .Chart.Name .Chart.Version) -}}
{{- end -}}

{{/*
Render the AppVersion as a valid Kubernetes label value.
*/}}
{{- define "labels.appVersion" -}}
{{- include "labels.sanitizeValue" .Chart.AppVersion -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "labels.common" -}}
application.giantswarm.io/team: {{ index .Chart.Annotations "io.giantswarm.application.team" | quote }}
{{ include "labels.monitoring" . }}
{{- end -}}

{{/*
Monitoring labels
*/}}
{{- define "labels.monitoring" -}}
app: {{ include "name" . | quote }}
{{ include "labels.selector" . }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
app.kubernetes.io/version: {{ include "labels.appVersion" . | quote }}
helm.sh/chart: {{ include "chart" . | quote }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "labels.selector" -}}
app.kubernetes.io/name: {{ include "name" . | quote }}
app.kubernetes.io/instance: {{ .Release.Name | quote }}
{{- end -}}

{{/*
Define image tag.
*/}}
{{- define "image.tag" -}}
{{- .Values.image.tag | default .Chart.AppVersion -}}
{{- end -}}

{{/*
Resolve the config audit report flag.
A little magic for handling defaulting with booleans https://github.com/helm/helm/issues/3308#issuecomment-701367019
*/}}
{{- define "exporter.configAuditReportsEnabled" -}}
{{- hasKey .Values.exporter.configAuditReports "enabled" | ternary .Values.exporter.configAuditReports.enabled true -}}
{{- end -}}

{{/*
Resolve a vulnerability report scanner flag.
*/}}
{{- define "exporter.vulnerabilityReports.scannerEnabled" -}}
{{- $enabled := false -}}
{{- if or (not (hasKey .Values.exporter.vulnerabilityReports "enabled")) (eq .Values.exporter.vulnerabilityReports.enabled true) -}}
  {{- if hasKey .Values.exporter.vulnerabilityReports "scanners" -}}
    {{- if and (hasKey .Values.exporter.vulnerabilityReports.scanners .scanner) (hasKey (index .Values.exporter.vulnerabilityReports.scanners .scanner) "enabled") -}}
      {{- $scannerConfig := index .Values.exporter.vulnerabilityReports.scanners .scanner -}}
      {{- $enabled = $scannerConfig.enabled -}}
    {{- end -}}
  {{- else if hasKey .Values.exporter.vulnerabilityReports "enabled" -}}
    {{- $enabled = .Values.exporter.vulnerabilityReports.enabled -}}
  {{- end -}}
{{- end -}}
{{- $enabled -}}
{{- end -}}

{{- define "exporter.vulnerabilityReports.trivyEnabled" -}}
{{- include "exporter.vulnerabilityReports.scannerEnabled" (dict "Values" .Values "scanner" "trivy") -}}
{{- end -}}

{{- define "exporter.vulnerabilityReports.kubescapeEnabled" -}}
{{- include "exporter.vulnerabilityReports.scannerEnabled" (dict "Values" .Values "scanner" "kubescape") -}}
{{- end -}}

{{/*
Pod scheduling constraints: renders the `nodeSelector` and `tolerations` fields,
merging the `architecture` shorthand into the explicit values for both.

Giant Swarm arm64 node pools carry a `kubernetes.io/arch=arm64:NoSchedule`
taint, so a pod bound to arm64 needs the node selector *and* the matching
toleration: with only the selector it stays Pending, with only the toleration
it may be scheduled onto any pool.

The toleration is therefore derived from the *effective* `kubernetes.io/arch`
selector, whichever of the two values set it, so pinning through `nodeSelector`
alone is as safe as pinning through `architecture`. It is skipped when an
explicit toleration already covers that taint, matching on the fields that
decide coverage rather than on the whole dict.

A `nodeSelector` that sets `kubernetes.io/arch` to something other than
`architecture` is a contradiction rather than a preference to arbitrate, so it
fails the render instead of silently discarding one of the two.
Emits nothing when unset, so rendered output is unchanged for existing users.
*/}}
{{- define "starboard-exporter.podScheduling" -}}
{{- $nodeSelector := deepCopy (.Values.nodeSelector | default dict) -}}
{{- $tolerations := .Values.tolerations | default list -}}
{{- with .Values.architecture -}}
{{- if and (hasKey $nodeSelector "kubernetes.io/arch") (ne (index $nodeSelector "kubernetes.io/arch") .) -}}
{{- fail (printf "architecture=%s conflicts with nodeSelector.%q=%s; set only one" . "kubernetes.io/arch" (index $nodeSelector "kubernetes.io/arch")) -}}
{{- end -}}
{{- $nodeSelector = merge (dict "kubernetes.io/arch" .) $nodeSelector -}}
{{- end -}}
{{- if eq (index $nodeSelector "kubernetes.io/arch" | default "") "arm64" -}}
{{- $tolerated := false -}}
{{- range $tolerations -}}
{{- if and (eq (.key | default "") "kubernetes.io/arch") (has (.effect | default "") (list "" "NoSchedule")) -}}
{{- if or (eq (.operator | default "Equal") "Exists") (eq (.value | default "") "arm64") -}}
{{- $tolerated = true -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- if not $tolerated -}}
{{- $tolerations = concat $tolerations (list (dict "key" "kubernetes.io/arch" "operator" "Equal" "value" "arm64" "effect" "NoSchedule")) -}}
{{- end -}}
{{- end -}}
{{- $scheduling := dict -}}
{{- if $nodeSelector -}}{{- $_ := set $scheduling "nodeSelector" $nodeSelector -}}{{- end -}}
{{- if $tolerations -}}{{- $_ := set $scheduling "tolerations" $tolerations -}}{{- end -}}
{{- with $scheduling }}{{ toYaml . }}{{ end -}}
{{- end -}}
