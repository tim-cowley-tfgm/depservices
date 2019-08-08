# circular-services

An AWS Lambda function that downloads a zipped TransXChange dataset from an 
AWS S3 bucket and then parses each TransXChange file to determine if it relates
to a circular service. If the service is a circular, the service description
is stored in a Redis cache; this can then be used as the destination for
circular services.

## Source data

TransXChange data is used as the data source.

## Redis data structure

The key for each service will consist of a concatenation of the
**operator code** - `Operators -> Licensed Operator -> OperatorCode` - and the
**service number** - `Services -> Service -> Lines -> Line -> LineName`.

The value will be the **service description** - 
`Services -> Service -> Description`.

```
SCMN12: Middleton - Moorclose circular
FMAN19: Middleton - Langley circular
MCTR232: Ashton - Hunts Cross - Broadoak circular
VISB500: Bolton Town Centre Metroshuttle
VISB525: Bolton - Halliwell - Hall i'th Wood circular
```

## Determining a circular service

We determine a circular service by performing a case-insensitive search on the
`Services -> Service -> Description` tag for relevant search terms. The Central
Data Maintenance team will ensure that descriptions for circular services are
appropriate and will modify the TransXChange files provided by operators if
the descriptions in these do not conform to our pattern.

We have explored other options for determining a circular service which do not
rely on manual interventions from the Central Data Maintenance team, 
e.g. checking if the origin and destination are the same; however, we found 
instances of circular services that run short or otherwise do not end at 
exactly the same location as they originate from; it also leaves us more 
vulnerable to inconsistencies in the data caused by the operators.

## Environment variables

* **CIRCULAR_SERVICES_REDIS_HOST** - The host for the _circular services_ Redis 
  cache; e.g. `localhost:6379`
* **CIRCULAR_SERVICES_REDIS_MAX_ACTIVE** _(optional)_ - The maximum number of 
  active connections allowed to the Redis host; defaults to `10`.
* **FLUSH_AFTER** _(optional)_ - The number of records to enqueue before sending
  the payload to the Redis host; defaults to `10000`.
* **SEARCH_TERMS** - A semi-colon separated list of search terms to use when
  determining a circular service; e.g. `circular;Metroshuttle`. These terms
  will replace those found in service descriptions, to ensure consistency with
  capitalisation, etc.
* **TXC_S3_BUCKET** - An AWS S3 bucket where the zipped TransXChange data
  will be stored; e.g. `data-surfacing`
* **TXC_S3_PREFIX** _(optional)_ - A prefix for the object in the S3 bucket; 
  e.g. `/departures-service/TransXChange/dev/`
