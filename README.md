# chalkular :vampire:

[ocular](https://github.com/crashappsec/ocular) 🤝 [chalk](https://github.com/crashappsec/chalk)

Chalkular is a service that will listen for requests to analyze container images (called "artifacts") and schedule Ocular pipelines.

## Getting started

1. Configure the intake methods you desire (below)
2. Create an `ArtifactMediaTypeMapping` resource in the namespace you want your scan to run in.
   This resource will tell chalkular which pipelines to create for a container image based on
   their media type. For example, the following mapping will start pipelines for standard
   docker images with profile `analyze` and the downloader `tar-docker` in the `scans` namespace:
	
	```yaml
	apiVersion: chalkular.ocular.crashoverride.run/v1beta1
	kind: ArtifactMediaTypeMapping
	metadata:
		name: docker-mapping
		namespace: scans
	spec:
		mediaTypes:
			- "application/vnd.oci.image.index.v1+json"
			- "application/vnd.oci.image.manifest.v1+json"
		profile:
			# this assumes 'analyze' exists in 'scans' namespace
			valueFrom:
				name: analyze
			# you can also specify a profile spec 
			# in 'value' and instead have chalkular manage
			# the profile for you
		downloader:
			# this assumes 'tar-docker' exists in 'scans' namespace
			valueFrom:
				name: tar-docker
			# you can also specify a downloader spec 
			# in 'value' and instead have chalkular manage
			# the downloader for you
			# value: ...
	```
3. Send an event to the intake method. The method will take in an OCI image reference and a namespace.
   Chalkular will read the image's media type and then look for artifact media type mappings in the given namespace that
   have the image's media type in the `mediaTypes` list. For each mapping that does, that mappings pipeline will be created.
4. Viewing the piplines in the namespace should reveal that one was created for the `docker-mapping` mapping

### Intake Methodsx

Chalkular supports the following intake methods for artifacts:

| Protocol | Notes                                                                                                                                                                                                                                                          |
| ======== | ======================================================================================================================================================================                                                                                         |
| `SQS`    | Chalkular will listen for messages from an SQS queue and when it recieves a message, will read the `namespace` and `image_uri` from the message attributes as strings. The SQS queue URL should be passed as the CLI argument `--sqs-queue-url`                |
| `HTTP`   | Chalkular will start a new webserver and listen for HTTP POST requests for the path `/chalkular/v1bet1/artifacts/analyze`. The port can be set by the CLI arg `--artifact-http-bind-addr`. NOTE: any service or ingress will need to be managed by the enduser |
	