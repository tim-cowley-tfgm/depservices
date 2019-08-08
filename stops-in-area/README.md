# stops-in-area

An AWS Lambda function that downloads the NaPTAN CSV dataset and stores the 
relationship between each stop and its stop area in a Redis database. (i.e.
which bus stand is in which bus station.)

The purpose of storing this data is to facilitate the 
[Ingester](../ingester/README.md) to cache responses for entire bus stations.

It is intended that this "stops-in-area" Lambda function is run infrequently,
e.g. nightly, on demand, or if the target Redis cache fails.

## Source data

The source data is derived from the `StopsInArea.csv` file in the NaPTAN CSV
dataset. Only the values in the first two columns are used.

## Redis data structure

The data will be stored with the stop ATCO codes as keys, with the stop area
name(s) as values:

```
1800BNIN0A1: 1800BNIN
1800BNIN0B1: 1800BNIN
...
1800BNIN0X1: 1800BNIN
1800BNIN0Y1: 1800BNIN
1800WNBS0A1: 180GWNBS
1800WNBS0B1: 180GWNBS
...
1800WNBS0T1: 180GWNBS
1800WNBS0U1: 180GWNBS
```

_Note: The NaPTAN dataset does not currently contain any duplicate stop ATCO 
codes; we do not check for or handle this_

## Environment variables

* **NAPTAN_CSV_DATA_SOURCE** - The location of the NaPTAN CSV dataset in 
  ZIP format; e.g. `http://naptan.app.dft.gov.uk/DataRequest/Naptan.ashx?format=csv`
* **NAPTAN_CSV_STOPS_IN_AREA_FILENAME** _(optional)_ - The name of the CSV file
  containing the data we need. Defaults to `StopsInArea.csv`.
* **NAPTAN_CSV_TIMEOUT** _(optional)_ - A timeout value in seconds for downloading the
  NaPTAN CSV dataset; defaults to `60`. _Note: the NaPTAN CSV dataset is
  approximately 30MB in size and doesn't appear to be hosted on a particularly
  fast connection.
* **STOPS_IN_AREA_REDIS_HOST** - The host for the _stops in area_ Redis cache; 
  e.g. `localhost:6379`
* **STOPS_IN_AREA_REDIS_MAX_ACTIVE** _(optional)_ - The maximum number of active
  connections allowed to the Redis host; defaults to `10`.
* **FLUSH_AFTER** _(optional)_ - The number of records to enqueue before sending
  the payload to the Redis host; defaults to `10000`.
