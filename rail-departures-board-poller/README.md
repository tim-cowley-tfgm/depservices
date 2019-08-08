# rail-departures

An AWS Lambda function to poll the National Rail Enquiries (NRE) 
[Live Departure Boards Web Service](http://lite.realtime.nationalrail.co.uk/openldbws/)
to retrieve information on upcoming departures for train stations in Greater
Manchester.

The function uses the LDBWS GetDepartureBoard endpoint to retrieve departures
for each station; it makes one request per station.

The data undergoes some minor transformation before being output to an
AWS SNS topic.

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

* **AWS_SNS_TOPIC_ARN** - The AWS SNS topic URN to publish departures to
* **NRE_OPENLDBWS_URL** - The URL for accessing the LDBWS
* **NRE_OPENLDBWS_ACCESS_TOKEN** - A token that authorises access to the LDBWS service;
  see the [OpenLDBWS troubleshooting guide](https://wiki.openraildata.com/index.php/OpenLDBWS_Troubleshooting)
  for more info on this
