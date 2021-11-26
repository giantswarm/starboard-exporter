# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2021-11-26

### Added

- Make target labels more easily configurable in `values.yaml`.

### Added

- Add configurable target labels.
- Add Grafana dashboard.
- Support custom labels for ServiceMonitor.

## [0.0.1] - 2021-11-18

### Added

- Add `image_vulnerabilities` metric per-CVE per-image and `image_vulnerabilities_count` metric for summaries.
- Add ServiceMonitor to scrape metrics.

[Unreleased]: https://github.com/giantswarm/starboard-exporter/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/giantswarm/starboard-exporter/compare/v0.0.1...v0.1.0
[0.0.1]: https://github.com/giantswarm/starboard-exporter/releases/tag/v0.0.1
