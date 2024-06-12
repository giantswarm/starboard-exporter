# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.11] - 2024-06-12

### Changed

- Update to go 1.22 and bump dependencies.

## [0.7.10] - 2024-05-07

### Changed

- Remove API check for `HorizontalPodautoscaler`.

## [0.7.9] - 2024-05-03

### Changed

- Switched API version from the `HorizontalPodAutoscaler` from `autoscaling/v2beta1` to `autoscaling/v1`.

## [0.7.8] - 2024-01-16

### Changed

- Switch PSP values from `psp.enabled` to `global.podSecurityStandards.enforced`.

## [0.7.7] - 2023-12-19

### Added

- Add a `scaledObject` resource for KEDA support.

## [0.7.6] - 2023-12-12

### Changed

- Update go to v1.21 and bump dependencies.

## [0.7.5] - 2023-12-06

### Changed

- Configure `gsoci.azurecr.io` as the default container image registry.

### Removed

- Stop pushing to `openstack-app-collection`.

## [0.7.4] - 2023-04-24

### Added

- Add icon.

## [0.7.3] - 2023-04-12

### Changed

- Removed `application.giantswarm.io/team` label from ServiceMonitor.

## [0.7.2] - 2023-02-27

### Changed

- Fix/template RoleBinding for deploying into namespaces other than the release namespace.

## [0.7.1] - 2023-01-25

### Added

- Adds `imagePullSecrets` to Chart.

## [0.7.0] - 2023-01-11

### Changed

- Replaces starboard library with trivy-operator library.
- Removes CIS benchmarks & reporting capabilities.

### Added

- Add Horizontal Pod Autoscaling based on Prometheus scrape times.

## [0.6.3] - 2022-12-02

## [0.6.2] - 2022-10-24

### Changed

- Fix schema type for tolerations ([#157](https://github.com/giantswarm/starboard-exporter/issues/157)).

## [0.6.1] - 2022-10-21

### Changed

- Make ServiceMonitor relabelings configurable and drop unhelpful pod, container, and service labels by default.
- Build with `app-build-suite`.
- Add `app-test-suite` basic smoke tests.

## [0.6.0] - 2022-09-16

### Added

- Add `podLabels` property to allow custom pod labels.

### Changed

- Disable reconciliation of CIS benchmark reports by default. These reports are temporarily removed from `trivy-operator`, to be reintroduced in the future. Reconciliation of CIS benchmarks produced by `starboard` is still supported by setting `exporter.CISKubeBenchReports.enabled: true` in the Helm values.

## [0.5.2] - 2022-09-09

### Added

- Make `interval` and `scrapeTimeout` configurable in the service monitor via `monitoring.serviceMonitor.interval` and `monitoring.serviceMonitor.scrapeTimeout`

## [0.5.1] - 2022-07-13

### Added

- Allow selectively enabling/disabling controllers for each report type.

## [0.5.0] - 2022-06-22

### Announcements

- **Important: the `latest` tag alias is being removed.** Some users have reported issues using the `latest` tag on our hosted registries (Docker Hub, Quay, etc.). We advise against using `latest` tags and don't use them ourselves, so this tag is not kept up to date. Please switch to using a tagged version. We will be removing the `latest` tag from our public registries in the near future to avoid confusion.

### Added

- Add missing monitoring options in the Helm chart values.yaml.
- Support sharding report metrics across multiple instances of the exporter.
- Set `runAsNonRoot` and use `RuntimeDefault` seccomp profile.
- Make replica count configurable in Helm values.
- Add configurable tolerations to Helm values.
- Reconcile and expose metrics for `CISKubeBenchReport` custom resources.

## [0.4.1] - 2022-04-26

### Added

- Spread (jitter) re-queueing of reports by +/- 10% by default to help smooth resource utilization.

## [0.4.0] - 2022-04-22

### Added

- Reconcile and expose metrics for `ConfigAuditReport` custom resources. **Requires Starboard v0.15.0 or above.**

## [0.3.3] - 2022-03-31

### Changed

- Build with [`architect`](https://github.com/giantswarm/architect) instead of [`app-build-suite`](https://github.com/giantswarm/app-build-suite) (reverts change from 0.3.2).

## [0.3.2] - 2022-03-28

### Added

- Add configurable nodeSelector to Helm values.

### Changed

- Build with [`app-build-suite`](https://github.com/giantswarm/app-build-suite) instead of [`architect`](https://github.com/giantswarm/architect).

## [0.3.1] - 2022-03-15

### Added

- Add NodeAffinity to run the exporter only on Linux Nodes with AMD64.

## [0.3.0] - 2022-02-14

### Added

- Add the `image_registry` label exposing the image registry.

### Changed

- Bump `golang`, `prometheus`, and `starboard` dependency versions.
- Update Grafana dashboard to use plugin version 8.3.2 and the new label.

## [0.2.1] - 2022-01-24

### Added

- Make pod annotations configurable.
- Bump `golang`, `prometheus`, and `starboard` versions.

## [0.2.0] - 2022-01-05

### Added

- Helm, add configurable container securityContext with secure defaults.

### Changed

- Bump `starboard`, `logr`, and `controller-runtime` dependency versions.
- Remove unneeded `releaseRevision` annotation from deployment.

### Fixed

- Helm, fix incomplete metric name in pods with high/critical CVEs panel

## [0.1.4] - 2021-12-14

### Changed

- Helm, remove unused RBAC config and add if for PSP and NetworkPolicy yaml.

## [0.1.3] - 2021-12-10

### Changed

- Make pod resource requests/limits configurable via helm values.

## [0.1.2] - 2021-11-29

### Changed

- Push images to Aliyun.
- Add `starboard-exporter` to AWS and Azure app collections.

## [0.1.1] - 2021-11-26

### Added

- Make target labels more easily configurable in `values.yaml`.

## [0.1.0] - 2021-11-26

### Added

- Add configurable target labels.
- Add Grafana dashboard.
- Support custom labels for ServiceMonitor.

## [0.0.1] - 2021-11-18

### Added

- Add `image_vulnerabilities` metric per-CVE per-image and `image_vulnerabilities_count` metric for summaries.
- Add ServiceMonitor to scrape metrics.

[Unreleased]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.11...HEAD
[0.7.11]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.10...v0.7.11
[0.7.10]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.9...v0.7.10
[0.7.9]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.8...v0.7.9
[0.7.8]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.7...v0.7.8
[0.7.7]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.6...v0.7.7
[0.7.6]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.5...v0.7.6
[0.7.5]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.4...v0.7.5
[0.7.4]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.3...v0.7.4
[0.7.3]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.2...v0.7.3
[0.7.2]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.6.3...v0.7.0
[0.6.3]: https://github.com/giantswarm/starboard-exporter/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/giantswarm/starboard-exporter/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.5.2...v0.6.0
[0.5.2]: https://github.com/giantswarm/starboard-exporter/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.4.1...v0.5.0
[0.4.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.3.3...v0.4.0
[0.3.3]: https://github.com/giantswarm/starboard-exporter/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/giantswarm/starboard-exporter/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.4...v0.2.0
[0.1.4]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/giantswarm/starboard-exporter/releases/tag/v0.0.1
