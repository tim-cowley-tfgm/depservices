# OPTIS Poller

An AWS Lambda function that makes a request to the OPTIS Stop Monitoring
request/response endpoint, filters out any non-departure records and 
publishes the useful data for any departures received to an AWS 
Simple Notification Service (SNS) topic.

## AWS Permissions

The Lambda function needs an IAM role which grants the following permissions:

```json
{
  "Action": [
    "SNS:Publish"
  ],
  "Effect": "Allow",
  "Resource": "arn:aws:sns:<region>:<account-id>:<topic-name>"
}
```

Where `<region>` is the AWS region (e.g. `eu-west-1`), `<account-id>` is the
AWS account ID and `<topic-name>` is the name given to the SNS topic.

## Environment

The following values need to be configured as environment variables:

* **AWS_SNS_TOPIC_ARN** - The AWS SNS topic ARN to publish departures to
* **OPTIS_STOP_MONITORING_REQUEST_URL** - The OPTIS endpoint to make requests to
* **OPTIS_TIMEOUT** _(optional)_ - The timeout in seconds for making a request
  and receiving a response from OPTIS. Defaults to `30`.
* **OPTIS_API_KEY** - The API key to access OPTIS
* **OPTIS_REQUESTOR_REF** - A requestor reference with permissions to access
  the resource on OPTIS
* **OPTIS_PREVIEW_INTERVAL** - An ISO8601 duration string specifying the 
  period of time we want departures for; e.g. `PT2H` means we want the 
  departures for the next two hours
* **OPTIS_MAXIMUM_STOP_VISITS** - A numeric value specifying the maximum number 
  of records that OPTIS should return; e.g. `200`

## Execution

The function requires an event payload containing the ATCO code of a bus
station; e.g.

```json
{
  "atcocode": "1800BNIN"
}
```

This will need to be configured with AWS CloudWatch Scheduled Events, or
a similar service.

The Lambda function should be executed independently per location and ideally
with a reasonable gap between requests, so that we don't flood OPTIS with 
multiple requests at the same time.

## Output Payload

The function will publish a payload containing a JSON representation of a 
[departures struct](../model/README.md) to the SNS topic.
