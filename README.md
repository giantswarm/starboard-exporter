[![CircleCI](https://circleci.com/gh/giantswarm/starboard-exporter.svg?style=shield)](https://circleci.com/gh/giantswarm/starboard-exporter)

# starboard-exporter

Exposes Prometheus metrics from [Starboard][starboard-upstream]'s `VulnerabilityReport`, `ConfigAuditReport`, and other custom resources (CRs).

## Metrics

This exporter exposes several types of metrics:

### CIS Benchmarks

#### Report Summary

A report summary series exposes the count of checks of each status reported in a given `CISKubeBenchReport`. For example:

```shell
starboard_exporter_ciskubebenchreport_report_summary_count{
    node_name="bj56o-master-bj56o-000000"
    status="FAIL"
    } 31
```

#### Section Summary

For slightly more granular reporting, a section summary series exposes the count of checks of each status reported in a given `CISKubeBenchSection`. For example:

```shell
starboard_exporter_ciskubebenchreport_section_summary_count{
    node_name="bj56o-master-bj56o-000000"
    node_type="controlplane"
    section_name="Control Plane Configuration"
    status="WARN"
    } 4
```

#### Result Detail

A CIS benchmark result info series exposes fields from each instance of an Aqua `CISKubeBenchResult`. For example:

```shell
starboard_exporter_ciskubebenchreport_result_info{
    node_name="bj56o-master-bj56o-000000"
    node_type="controlplane"
    pod="starboard-exporter-859955f485-cwkj6"
    section_name="Control Plane Configuration"
    test_desc="Client certificate authentication should not be used for users (Manual)"
    test_number="3.1.1"
    test_status="WARN"
    } 1
```

### Vulnerability Reports

#### Report Summary

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

#### Vulnerability Details

A "detail" or "vulnerability" series exposes fields from each instance of an Aqua `Vulnerability`. The value of the metric is the `Score` for the vulnerability. For example:

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
    vulnerability_title="openssl: Read buffer overruns processing ASN.1 strings",
    vulnerable_resource_name="libssl1.1"
    } 7.4
```

This indicates that the vulnerability with the id `CVE-2021-3712` was found in the `giantswarm/starboard-operator` image in the `demo` namespace, and it has a CVSS 3.x score of 7.4.

An additional series would be exposed for every combination of those labels.

### Config Audit Reports

#### Report Summary

A summary series exposes the count of checks of each severity reported in a given `ConfigAuditReport`. For example:

```shell
starboard_exporter_configauditreport_resource_checks_summary_count{
  resource_name="replicaset-chart-operator-748f756847",
  resource_namespace="giantswarm",
  severity="LOW"
  } 7
```

#### A Note on Cardinality

For some use cases, it is helpful to export additional fields from `VulnerabilityReport` CRs. However, because many fields contain unbounded arbitrary data, including them in Prometheus metrics can lead to extremely high cardinality. This can drastically impact Prometheus performance. For this reason, we only expose summary data by default and allow users to opt-in to higher-cardinality fields.

### Sharding Reports

In large clusters or environments with many reports and/or vulnerabilities, a single exporter can consume a large amount of memory, and Prometheus may need a long time to scrape the exporter, leading to scrape timeouts. To help spread resource consumption and scrape effort, `starboard-exporter` watches its own service endpoints and will shard metrics for all report types across the available endpoints. In other words, if there are 3 exporter instances, each instance will serve roughly 1/3 of the metrics. This behavior is enabled by default and does not require any additional configuration. To use it, simply change the number of replicas in the Deployment. However, you should read the section on cardinality and be aware that consuming large amounts of high-cardinality data can have performance impacts on Prometheus.

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

The same can be done for CIS Benchmark Results. To enable an additional detail series *per CIS Benchmark Result*, use the `--cis-detail-report-labels` flag to specify which labels should be exposed. For example:

```shell
# Expose only section_name, test_name and test_status
--cis-detail-report-labels=section_name,test_name,test_status

# Run with (almost) all fields exposed as labels.
--cis-detail-report-labels=all
```

CIS detail target labels can also be set via Helm values:

```yaml
exporter:
  CISKubeBenchReports:
    targetLabels:
      - node_name
      - node_type
      - section_name
      - test_name
      - test_status
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

## Scaling for Prometheus scrape timeouts

When exporting a large volume of metrics, Prometheus might time out before retrieving them all from a single exporter instance. It is possible to automatically scale the number of exporters to keep the scrape time below the configured timeout. To enable HPA scaling based on Prometheus metrics, [here](./docs/custom_metrics_hpa.md)
