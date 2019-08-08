# Ingester

An AWS Lambda function that can be subscribed to SNS topics that provide a
[Departures](../model/internal.go) payload. The ingester uses this payload
to create or update the departures cache, which can then be queried by other
services.

The ingester performs a number of functions:

* Updates the destination name to a more meaningful value from either the 
  [circular services](../circular-services/README.md) cache or the
  [locality names](../locality-names/README.md) cache;
* Sorts the departures into a sensible order;
* Caches departures for the stop for quick access from the 
  [presenter](../presenter/README.md); and
* When the departures are for stop(s) in a 
  [stop area](../stops-in-area/README.md), combines the departures for stops 
  and stores them in the cache, too.

## Triggers

The function is intended to be triggered whenever data is published to a
subscribed SNS topic.

## Incoming payload

The function expects to receive a JSON payload containing a 
[internal model](../model/README.md).

## Output

The output is stored in a Redis database with the location ATCO code as the
key.

## Environment

The function requires the following environment setup:

* **DEPARTURES_REDIS_HOST**: The address to use to connect to the Redis
  _departures_ cache; e.g. `localhost:6379`
* **LOCALITY_NAMES_REDIS_HOST**: The address to use to connect to the Redis
  _locality names_ cache; e.g. `localhost:6379`
* **STOPS_IN_AREA_REDIS_HOST**: The address to use to connect to the Redis
  _stops in area_ cache; e.g. `localhost:6379`
* **CIRCULAR_SERVICES_REDIS_HOST**: The address to use to connect to the Redis
  _stops in area_ cache; e.g. `localhost:6379`
