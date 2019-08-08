# Presenter

An AWS Lambda function that receives a request for a location and retrieves
the data for that location from the Redis cache, returning a JSON payload. 

The function transforms data so that it is useful for presentation.


## Departure times

Departure time information is combined; if real-time departure information is 
available, it is presented in preference to a scheduled departure time.

Real-time departures are presented as a countdown in minutes, rounded down to 
the nearest minute; e.g. `1 min`, `2 mins`. Services that are expected to 
arrive within one minute are presented as `Approaching`.

Where only scheduled departure information is available, the time is presented
as a time of day in 24-hour clock format, again rounded down to the nearest
minute; e.g. `14:26`


## Triggers

The function is intended to be triggered whenever a request is made from the
AWS API Gateway.

## Incoming payload

The function expects to receive an AWS API Gateway Proxy Request, containing an 
`atcocode` and an optional `top` value:

```json
{
  "queryStringParameters": {
    "atcocode": "1800BNIN0C1"
  }
}
```

The presenter will retrieve a maximum of 10 departures if no `top` value is
provided.

```json
{
  "queryStringParameters": {
    "atcocode": "1800BNIN0C1",
    "top": 20
  }
}
```
## Output

The presenter returns an [output model](../model/output.go)

## Environment

The function requires the following environment setup:

* **DEPARTURES_REDIS_HOST**: The address to use to connect to Redis
  _departures_ cache; e.g. `localhost:6379`
