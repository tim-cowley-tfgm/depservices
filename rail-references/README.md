# rail-references

An AWS Lambda function that downloads the NaPTAN CSV dataset and stores the 
relationship between National Rail Computer Reservation System codes (CRS) and 
its NaPTAN ATCO code in a Redis database.

The purpose of storing this data is to facilitate the 
[Rail Departures Board Poller](../rail-departures-board-poller/README.md) 
in appending ATCO code identifiers to data retrieved from National Rail Enquiries.

**This function is not currently used.** Instead, we are using the
[GetAtcoCode](../nationalrail/README.md) function to get the relationship
between CRS codes and ATCO codes.

If this function does come into use, it is intended that it is run infrequently,
e.g. nightly, on demand, or if the target Redis cache fails.

## Source data

The source data is derived from the `RailReferences.csv` file in the NaPTAN CSV
dataset. Only the values in the first and third columns is used
(`AtcoCode` and `CrsCode`).

## Redis data structure

The data will be stored with the CRS Codes as keys, with the ATCO codes as 
values:

```
ADK: 9100ARDWICK
GTY: 9100GATLEY
LVM: 9100LVHM
MAN: 9100MNCRPIC
MAU: 9100MLDTHRD
```

## Environment variables

* **NAPTAN_CSV_DATA_SOURCE** - The location of the NaPTAN CSV dataset in 
  ZIP format; e.g. `http://naptan.app.dft.gov.uk/DataRequest/Naptan.ashx?format=csv`
* **NAPTAN_CSV_RAIL_REFERENCES_FILENAME** _(optional)_ - The name of the CSV file
  containing the data we need. Defaults to `RailReferences.csv`.
* **NAPTAN_CSV_TIMEOUT** _(optional)_ - A timeout value in seconds for downloading the
  NaPTAN CSV dataset; defaults to `60`. _Note: the NaPTAN CSV dataset is
  approximately 30MB in size and doesn't appear to be hosted on a particularly
  fast connection.
* **RAIL_REFERENCES_REDIS_HOST** - The host for the _rail references_ Redis cache; 
  e.g. `localhost:6379`
* **RAIL_REFERENCES_REDIS_MAX_ACTIVE** _(optional)_ - The maximum number of active
  connections allowed to the Redis host; defaults to `10`.
* **FLUSH_AFTER** _(optional)_ - The number of records to enqueue before sending
  the payload to the Redis host; defaults to `10000`.
