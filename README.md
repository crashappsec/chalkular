# chalkular :vampire:

[ocular](https://github.com/crashappsec/ocular) 🤝 [chalk](https://github.com/crashappsec/chalk)

Chalkular is a service that will listen for requests to analyze container images (called "artifacts") and schedule Ocular pipelines.

Chalkular provides a set of [intake methods](#intake-methods) for the images to schedule pipelines for (see below for the full list).
When a new image is received via an intake method, the contoller will see if the media type of the image is referenced in
a `MediaTypePolicy`, a resource managed by Chalkular specifies a pipeline that should be created when an image with a matching
media type is received.

## Getting started

1. Configure the intake methods you desire (below)
2. Create an `MediaTypePolicy` resource in the namespace you want your scan to run in.
   This resource will tell chalkular which pipelines to create for a container image based on
   their media type. For example, the following mapping will start pipelines for standard
   docker images with profile `analyze` and the cluster downloader `chalkular-artifacts` in the `scans` namespace:
	
    ```yaml
    apiVersion: chalk.ocular.crashoverride.run/v1beta1
    kind: ChalkReportPolicy
    metadata:
      name: docker-images
	  namespace: scan
    spec:
      matchCondition: 'chalk._OP_ARTIFACT_TYPE == "Docker Image"'
      extraction:
        target: |
          {
	        'identifier': chalk._X_OCULAR_TARGET_IDENTIFIER,
	        'version': chalk._X_OCULAR_TARGET_VERSION
          }
        downloaderParams: |
          { 'MEDIA_TYPE': chalk._X_OCULAR_MEDIA_TYPE }
        profileParams: |
          {
            'RUN_SECRETSCANNER': report.X_CHALK_PROFILE_CONFIG.runSecretScanner ? "1" : "",
            'RUN_SBOM': report.X_CHALK_PROFILE_CONFIG.runSbomTools ? "1" : "",
            'RUN_SAST': report.X_CHALK_PROFILE_CONFIG.runSastTools ? "1" : ""
          }
      pipelineTemplate:
        profileRef:
		  name: analyze # this assumes 'analyze' exists in the 'scan' namespace
		  namespace: scan
        downloaderRef:
          name: chalkular-artifacts # this is bundled with chalkular install
          kind: ClusterDownloader
        # other optional specifications for pipeline
    ```
3. Send an event to the intake method. The method will take in an OCI image reference and a namespace.
   Chalkular will read the image's media type and then look for artifact media type mappings in the given namespace that
   have the image's media type in the `mediaTypes` list. For each mapping that does, that mappings pipeline will be created.
4. Viewing the piplines in the namespace should reveal that one was created for the `docker-mapping` mapping

### Intake Methods

Chalkular supports the following intake methods for artifacts:

| Method   | Notes                                                                                                                                                                                                                                                          |
|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `SQS`    | Chalkular will listen for messages from an SQS queue and when it recieves a message, will read the `namespace` and `image_uri` from the message attributes as strings. The SQS queue URL should be passed as the CLI argument `--sqs-queue-url`                |
| `HTTP`   | Chalkular will start a new webserver and listen for HTTP POST requests for the path `/chalkular/v1bet1/artifacts/analyze`. The port can be set by the CLI arg `--artifact-http-bind-addr`. NOTE: any service or ingress will need to be managed by the enduser |

