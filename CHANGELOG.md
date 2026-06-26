# Chalkular Release Notes
<!-- https://keepachangelog.com -->
# [v0.0.6](https://github.com/crashappsec/chalkular/releases/tag/v0.0.6) - **June 26th, 2026**

### Added

- OpenMetrics for SQS and Scheduler
  - SQS stats for messages read, errors and latency
  - Scheduler stats for pipelines created and errors

### Changed

- upgrade ocular to v0.3.3

# [v0.0.5](https://github.com/crashappsec/chalkular/releases/tag/v0.0.5) - **June 17th, 2026**

### Added

- Downloader now supports build context images
  - Triggered if the OCI image contains artifact type of `application/vnd.crashoverride.chalk.build-context.v1`
  - Writes build context to target directory

- Uploader supports ocular ingestion
  - Triggered if the ingestion host (parameter `INGEST_HOST`) and ingestion token (from secret) are specified

### Fixed

- Increase unit test coverage

# [v0.0.4](https://github.com/crashappsec/chalkular/releases/tag/v0.0.4) - **May 27th, 2026**

### Added

- Specify the max amount of pipelines that can be generated from a report via `--max-pipelines-per-policy`
- Specify a threshold of active pipelines to begin rejecting new reports from being scheduled via `--reject-report-pipeline-threshold`

# [v0.0.3](https://github.com/crashappsec/chalkular/releases/tag/v0.0.3) - **May 22nd, 2026**

### Added

- `forEach` CEL expresion was added to `ChalkReportPolicy` resource to allow the user to spawn multiple pipelines per report
  - The expression should return a list of values, and the policy will be evaluated for each with the value to `each`
  
- Added support for selecting how SQS should be parsed for a chalk report
  - `message-body` indicates the message body of the SQS event should be parsed as a chalk report
  - `s3-event` indicates the SQS event follows the structure of an S3 event notication, where the object contains the chalk report.

### Removed

- Chalk report policies are no longer evaluted per chalk mark, instead it is evaluated once per report.

# [v0.0.2](https://github.com/crashappsec/chalkular/releases/tag/v0.0.2) - **May 1st, 2026**

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


# [v0.0.1](https://github.com/crashappsec/chalkular/releases/tag/v0.0.1) - **February 10th, 2026**

### Added

- `MediaTypePolicy` which specifies a list of media types for images and the template for pipelines to create
- `HTTP` intake method for images to scan
- `SQS` intake method for images to scan
