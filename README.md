# chalkular :vampire:

[ocular](https://github.com/crashappsec/ocular) 🤝 [chalk](https://github.com/crashappsec/chalk)

Chalkular is a service that will listen for chalk reports and schedule Ocular pipelines.

Chalkular provides a set of [intake methods](#intake-methods) for reports to schedule pipelines for (see below for the full list).
When a new report is received via an intake method the contoller will parse the report and then check if any `ChalkReportPolicy`
matches the chalk mark and if so, templates and schedules an Ocular pipeline for each.

## Getting started

1. Configure the intake methods you desire (below)
2. Create an `ChalkReportPolicy` resource in the namespace you want your scan to run in.

   This resource will tell chalkular which pipelines to create for chalk reports that it detects.
   For example, the following mapping will start pipelines for standard
   docker images with profile `analyze` and the cluster downloader `chalkular-artifacts` in the `scans` namespace:
    ```yaml
    apiVersion: chalk.ocular.crashoverride.run/v1beta1
    kind: ChalkReportPolicy
    metadata:
      name: docker-images
      namespace: scan
    spec:
      matchCondition: 'report._CHALKS.any(c, c._OP_ARTIFACT_TYPE == "Docker Image")'
      extraction:
        # We can iterate over a collection of extracted objects, here 
        # we will iterate over all chalk marks that areof type  "Docker Image"
		forEach: 'report._CHALKS.filter(c, c._OP_ARTIFACT_TYPE == "Docker Image")'
        # If targets is a list, a pipeline will be started for each
        # Otherwise if it is a single struct, only 1 target will be started
        targets: |
          # here will we pull the key '_OCULAR_IMAGE_REPO' as the identifier
          # and '_OCULAR_IMAGE_SHA' as version for each chalk mark in the report
          each.map(c,
           {
             'identifier': c._OCULAR_IMAGE_REPO,
             'version': c._OCULAR_IMAGE_VERSION
           })
        downloaderParams: |
          { 'MEDIA_TYPE': each._X_OCULAR_MEDIA_TYPE }
        profileParams: |
          {
            'RUN_SECRETSCANNER': report.X_CHALK_PROFILE_CONFIG.runSecretScanner ? "1" : "",
            'RUN_SBOM': report.X_CHALK_PROFILE_CONFIG.runSbomTools ? "1" : "",
            'RUN_SAST': report.X_CHALK_PROFILE_CONFIG.runSastTools ? "1" : ""
          }
      pipelineTemplate:
        profileRef:
          name: analyze # this assumes 'analyze' exists in the 'scan' namespace
        downloaderRef:
          name: chalkular-artifacts # this is bundled with chalkular install
          kind: ClusterDownloader
        # other optional specifications for pipeline
    ```
    The `matchCondition` field should be a [CEL](https://cel.dev) expression that evalautes to a boolean
    indicating if for the chalk mark a pipeline should be created. If true, the `extraction` fields are evalauted
    using CEL as well. `target` should return a string->string map with two fields `identifier` and `version` which will
    then be set as the target identifier and version.
    The `downloaderParams` and `profileParams` fields should also return a string->string map but each key/value pair will
    be set as the downloader parametes and profile parameters of the pipeline.
    The expressions will have the standard CEL definitions, [cel-go extenstions](https://github.com/google/cel-go/tree/HEAD/ext) and additionally the variable `report`
    which is the full JSON report that was recieved.
3. Send a chalk report to the intake method. The Chalkular controller will process the chalk report,
   and will run the `matchCondition` for all `ChalkReportPolicies`.
   Any that return true will have a pipeline created to scan it.
4. Monitor created pipelines

### Chalk Report Intake

The Chalkular controller supports various methods of receiving chalk reports.
They are documented below:


| Method | Notes                                                                                                                                                                                                                                                                                                                                    | How to Configure                                                                                                                                                                                                                                   |
|--------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `SQS`  | Chalkular will listen for messages from an SQS queue and when it recieves a message, ~~it will read the chalk report from the payload~~ A chalk report is too large for SQS payload, this will be switched to read from the CO API or via S3 link. Credentials will be read from standard AWS SDK methods (`AWS_CONFIG` or Metadata URL) | The SQS queue URL should be passed as the CLI argument `--sqs-queue-url`. Additionally a "parser" should be specified with `--sqs-parser`, either `s3-event` for S3 notification events, or `message-body` to parse directly from the message body |
| `HTTP` | Chalkular will start a new webserver and listen for HTTP `POST` requests for the path `/api/v1beta1/report`, where the body should be the JSON chalk report. The user will need to supply an Bearer token for a kubernetes user with permission for `post` on the path `/api/v1beta1/report`.                                            | The port can be set by the CLI arg `--report-http-bind-addr`. NOTE: any service or ingress will need to be managed by the enduser                                                                                                                  |


