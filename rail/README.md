# rail-ingester

An AWS Lambda function that can be subscribed to SNS topics that provide a 
National Rail Station Board payload. The ingester uses this payload
to create or update the departures cache, which can then be queried by other
services.

The rail ingester:

* Appends the location atcocode to the data;
* Removes expired data; and
* Caches departures for the stop for quick access from the 
  [presenter](../presenter/README.md)

## Triggers

The function is intended to be triggered whenever data is published to a
subscribed SNS topic.

## Incoming payload

The function expects to receive a JSON payload containing a National Rail
Station Board.

## Output

The output is stored in a Redis database with the location ATCO code as the
key.

## Environment

The function requires the following environment setup:

* **DEPARTURES_REDIS_HOST**: The address to use to connect to the Redis
  _departures_ cache; e.g. `localhost:6379`
