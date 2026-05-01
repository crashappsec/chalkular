# Chalkular Release Notes
<!-- https://keepachangelog.com -->

# [v0.0.2](https://github.com/crashappsec/ocular/releases/tag/v0.0.2) - **May 1st, 2026**

### Added

- `ChalkReportPolicy` which can specify [CEL](https://cel.dev) expressions for templating 
  pipelines for chalk marks recieved via a chalk report.
	  - The policy will be evaluated on each chalkmark present in the `_CHALKS` key
	  in the report.
- HTTP Endpoint `/api/v1beta1/report` which can accept a chalk report upload
  and will send it to the chalk report policy evaluator to create pipeline

### Removed

- `MediaTypePolicy` is removed in favor of `ChalkReportPolicy`
- HTTP endpoint `/chalkular/v1beta1/artifacts` removed in favor of `/api/v1beta1/report` endpoint


# [v0.0.1](https://github.com/crashappsec/ocular/releases/tag/v0.0.1) - **February 10th, 2026**

### Added

- `MediaTypePolicy` which specifies a list of media types for images and the template for pipelines to create
- `HTTP` intake method for images to scan
- `SQS` intake method for images to scan
