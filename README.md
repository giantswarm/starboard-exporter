[![CircleCI](https://circleci.com/gh/giantswarm/starboard-exporter.svg?style=shield)](https://circleci.com/gh/giantswarm/starboard-exporter)

# starboard-exporter

Exposes Prometheus metrics from [Starboard][starboard-upstream]'s `VulnerabilityReport` custom resources (CRs).

## Metrics

This exporter exposes two types of metrics:

### Summary

A summary series exposes the count of CVEs of each severity reported in a given `VulnerabilityReport`. For example:

```shell
starboard_exporter_vulnerabilityreport_image_vulnerability_severity_count{
    image_digest="",
    image_namespace="demo",
    image_registry="quay.io",
    image_repository="giantswarm/starboard-operator",
    image_tag="0.11.0",
    report_name="replicaset-starboard-app-6894945788-starboard-app",
    severity="MEDIUM"
    } 4
```

This indicates that the `giantswarm/starboard-operator` image in the `demo` namespace contains 4 medium-severity vulnerabilities.

### Detail / Vulnerability

A detail or vulnerability series exposes fields from each instance of an Aqua `Vulnerability`. The value of the metric is the `Score` for the vulnerability. For example:

```shell
starboard_exporter_vulnerabilityreport_image_vulnerability{
    fixed_resource_version="1.1.1l-r0",
    image_digest="",
    image_namespace="demo",
    image_registry="quay.io",
    image_repository="giantswarm/starboard-operator",
    image_tag="0.11.0",
    installed_resource_version="1.1.1k-r0",
    report_name="replicaset-starboard-app-6894945788-starboard-app",
    severity="HIGH",
    vulnerability_id="CVE-2021-3712",
    vulnerability_link="https://avd.aquasec.com/nvd/cve-2021-3712",
    vulnerability_title="openssl: Read buffer overruns processing ASN.1 strings",vulnerable_resource_name="libssl1.1"
    } 7.4
```

This indicates that the vulnerability with the id `CVE-2021-3712` was found in the `giantswarm/starboard-operator` image in the `demo` namespace, and it has a CVSS 3.x score of 7.4.

An additional series would be exposed for every combination of those labels.

#### A Note on Cardinality

For some use cases, it is helpful to export additional fields from `VulnerabilityReport` CRs. However, because many fields contain unbounded arbitrary data, including them in Prometheus metrics can lead to extremely high cardinality. This can drastically impact Prometheus performance. For this reason, we only expose summary data by default and allow users to opt-in to higher-cardinality fields.

### One vulnerabilityreport per deployment

By default, Starboard generates a `VulnerabilityReport` per ReplicaSet in a Deployment.
This can cause confusion because vulnerabilities are still reported for Pods which no longer exist, i.e. you fix a CVE in your latest Deployment but the number of CVEs per Deployment stays the same in your metrics.

As of Starboard v0.14.0, the environment variable `OPERATOR_VULNERABILITY_SCANNER_SCAN_ONLY_CURRENT_REVISIONS` can be enabled to only generate a `VulnerabilityReport` from the latest ReplicaSet in the Deployment.

Check the [Starboard configuration docs][starboard-config] for more information.

## Customization

Summary metrics of the format described above are always enabled.

To enable an additional detail series *per Vulnerability*, use the `--target-labels` flag to specify which labels should be exposed. For example:

```shell
# Expose only select image and CVE fields.
--target-labels=image_namespace,image_repository,image_tag,vulnerability_id

# Run with (almost) all fields exposed as labels, if you're feeling really wild.
--target-labels=all
```

Target labels can also be set via Helm values:

```yaml
exporter:
  vulnerabilityReports:
    targetLabels:
      - image_namespace
      - image_repository
      - image_tag
      - vulnerability_id
      - ...
```

[starboard-upstream]: https://github.com/aquasecurity/starboard
[starboard-config]: https://github.com/aquasecurity/starboard/blob/main/docs/operator/configuration.md

## Helm

How to install the starboard-exporter using helm:

```shell
helm repo add giantswarm https://giantswarm.github.io/giantswarm-catalog
helm repo update
helm upgrade -i starboard-exporter --namespace <starboard namespace> giantswarm/starboard-exporter
```
