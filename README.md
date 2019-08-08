# departures-service

A collection of microservices to surface scheduled and real-time departure data
for use through an API gateway.

## Bus departures

### OPTIS Poller

An AWS Lambda function that retrieves scheduled and real-time departure data 
from OPTIS using the **request/response** methodology. Once the data has been 
retrieved, it filters out any non-departure records and publishes the useful 
data to an AWS SNS topic, which other services can then subscribe to.

[More information](optis-poller/README.md)

### Ingester

An AWS Lambda function that takes data produced by the OPTIS Poller and
stores it on a Redis cache.

[More information](ingester/README.md)

### Presenter

An AWS Lambda function that reads data produced by the Ingester from the Redis 
cache, filtering out any expired records, before returning it in a simplified
JSON format.

[More information](presenter/README.md)

### Stops In Area

An AWS Lambda function that downloads the NaPTAN CSV dataset and stores the
relationship between stops and stop areas in a Redis cache. (i.e. which stands 
are in which bus station)

[More information](stops-in-area/README.md)

### Locality Names

An AWS Lambda function that downloads the NaPTAN CSV dataset and stores the
relationship between stops and locality names in a Redis cache.

[More information](locality-names/README.md)

### Circular Services

An AWS Lambda function that downloads the zipped TransXChange dataset from
AWS S3 and stores the service description for circular services in a Redis 
cache.

[More information](circular-services/README.md)

### Rail References

An **unused** AWS Lambda function that downloads the NaPTAN CSV dataset and 
stores the relationship between National Rail CRS codes and ATCO codes in a 
Redis cache.

[More information](rail-references/README.md)

### Rail Departures Board Poller

An AWS Lambda function that retrieves rail departure boards for a provided 
station. Once the data has been retrieved, it undergoes some transformation
before being published to an AWS SNS topic, which other services can then 
subscribe to.

[More information](rail-departures-board-poller/README.md)

### Rail Ingester

An AWS Lambda function that takes data produced by the Rail Departures Board
Poller and stores it on a Redis cache.

[More information](rail-ingester/README.md)

## Logging

The logging implementation consists of a simple wrapper around the default Go
log package. The wrapper adds functions for debug output, which can be 
enabled by adding a build flag.

[More information](dlog/README.md)

## Deployment

Automatically deployed to Development than choose to deploy onto NFT and PRD
