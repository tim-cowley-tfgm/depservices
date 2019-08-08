# locality-names

An AWS Lambda function that downloads the NaPTAN CSV dataset and stores the 
relationship between each stop and its locality in a Redis database.

The purpose of storing this data is to facilitate the 
[Ingester](../ingester/README.md) in replacing service destinations with a more
meaningful value (e.g. replacing `Turning Circle` with `Oldhams Estate`).

It is intended that this "locality-names" Lambda function is run infrequently,
e.g. nightly, on demand, or if the target Redis cache fails.

## Source data

The source data is derived from the `Stops.csv` file in the NaPTAN CSV
dataset. Only the values in the first and the nineteenth columns is used
(`ATCOcode` and `LocalityName`).

## Redis data structure

The data will be stored with the stop ATCO codes as keys, with the locality 
names as values:

```
1800SB12331: Wythenshawe
1800SB12361: Moss Side
1800SB12371: Moss Side
1800SB14581: Northenden
1800SB14591: Northenden
1800SB15161: Woodhouse Park
1800SB15211: East Didsbury
```

## Environment variables

* **NAPTAN_CSV_DATA_SOURCE** - The location of the NaPTAN CSV dataset in 
  ZIP format; e.g. `http://naptan.app.dft.gov.uk/DataRequest/Naptan.ashx?format=csv`
* **NAPTAN_CSV_STOPS_FILENAME** _(optional)_ - The name of the CSV file
  containing the data we need. Defaults to `Stops.csv`.
* **NAPTAN_CSV_TIMEOUT** _(optional)_ - A timeout value in seconds for downloading the
  NaPTAN CSV dataset; defaults to `60`. _Note: the NaPTAN CSV dataset is
  approximately 30MB in size and doesn't appear to be hosted on a particularly
  fast connection.
* **LOCALITY_NAMES_REDIS_HOST** - The host for the _stops in area_ Redis cache; 
  e.g. `localhost:6379`
* **LOCALITY_NAMES_REDIS_MAX_ACTIVE** _(optional)_ - The maximum number of active
  connections allowed to the Redis host; defaults to `10`.
* **FLUSH_AFTER** _(optional)_ - The number of records to enqueue before sending
  the payload to the Redis host; defaults to `10000`.
