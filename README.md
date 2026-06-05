# AWS S3 CCF Plugin

This plugin collects read-only Amazon S3 bucket configuration, evaluates CCF Rego policy bundles, and emits evidence back through the CCF agent.

## Supported resource families

The collector can evaluate policies for:

- S3 buckets

## How it fits in CCF

The CCF agent starts this binary through HashiCorp go-plugin, passes configuration and policy paths over gRPC, and receives generated evidence through the runner callback. This repository does not call the CCF API directly.

During plugin initialisation, subject templates and risk templates declared by the configured policy bundle are registered through the agent API helper.

## Default policy bundle mapping

| Repository | Behavior | Primary input |
| --- | --- | --- |
| plugin-aws-s3-policies | bucket | input.bucket + input.bucket_context |

## Configuration

The plugin expects:

- AWS credentials through the default AWS SDK credential chain
- target regions from config.regions or config.region
- AWS_REGION as a fallback when plugin config does not provide a region

Any agent-supplied policy_data is passed through to Rego as data.*.

## Data collected

For each in-scope bucket, the plugin collects and normalises:

- bucket name, ARN, creation date, and home region
- bucket tags
- server-side encryption settings
- public access block settings
- bucket policy document, policy hash, and public policy status
- ownership controls
- versioning and MFA delete status
- object lock configuration
- lifecycle rules and retention summary
- replication configuration
- server access logging configuration
- static website hosting configuration

The policy input contains both the raw bucket resource under input.bucket and the evaluated context under input.bucket_context.

## Development

Run the local test suite with:

~~~shell
go test ./...
~~~

Or use the Makefile wrapper:

~~~shell
make test
~~~

Build the plugin binary with:

~~~shell
make build
~~~

This writes the compiled plugin to dist/plugin.

## Related repositories

- [plugin-aws-s3-policies](https://github.com/compliance-framework/plugin-aws-s3-policies)
