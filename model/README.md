# Model

Data structures for use across the service.

## Internal

An array of departure items. Each departure item contains:

* **recordedAtTime** - An IS08601 timestamp for when the data was created;
  can be used to adjust the `aimedDepartureTime` and `expectedDepartureTime`
  values when the data is consumed by another service at a later point in time
* **journeyType** - the type of journey; e.g. `bus`; `train`; `tram`
* **journeyRef** - a unique reference for the record at the location; can be
  used to identify a record in a dataset for update/removal
* **aimedDepartureTime** - An ISO8601 timestamp for when the service is
  scheduled to depart from the location
* **expectedDepartureTime** - An ISO8601 timestamp for when the service is
  expected to depart from the location, based on real-time data. The value will
  be `null` if no real-time data is available.
* **status** - The status for the departure; e.g. `On time`; `Delayed`; 
  `Cancelled`; `12:34`
* **locationAtcocode** - The ATCO code identifier for the location, based on
  the NAPTaN dataset
* **stand** - The stand/platform identifer for the location, if applicable. 
  Set to `null` where there is no stand identifier.
* **destinationAtcocode** - The ATCO code identifier for the destination, based
  on the NAPTaN dataset
* **destination** - The display name of the destination
* **serviceNumber** - The bus service number
* **operatorCode** - The national operator code

Example JSON payload:

```json
{
  "departures": [
    {
      "recordedAtTime": "2019-05-08T23:29:46+01:00",
      "journeyRef": "2019-05-08_1234",
      "aimedDepartureTime": "2019-05-08T23:34:00+01:00",
      "expectedDepartureTime": "2019-05-08T23:35:24+01:00",
      "locationAtcocode": "1800BNIN0C1",
      "destinationAtcocode": "1800WA12481",
      "serviceNumber": "123",
      "operatorCode": "ANWE" 
    },
    {
      "recordedAtTime": "2019-05-08T23:29:46+01:00",
      "journeyRef": "2019-05-08_1235",
      "aimedDepartureTime": "2019-05-08T23:37:00+01:00",
      "expectedDepartureTime": null,
      "locationAtcocode": "1800BNIN0D1",
      "destinationAtcocode": "1800BYIC0B1",
      "serviceNumber": "456",
      "operatorCode": "FMAN" 
    }
  ]
}
```

### Sorting

#### ByDepartureTime

`sort.Sort(ByDepartureTime(<[]Departure>))`

Sorts departures by departure time (using the `expectedDepartureTime`
if available, otherwise using the `aimedDepartureTime`). If these are equal,
departures are then sorted by their service number; initially by their prefix
(services with no prefix are positioned before those with a prefix), then by
ascending digits (e.g. service `12` is positioned before service `123`), then
finally by the suffix (services with no suffix are positioned before those with
a suffix). If there is still no differentiation, the departures are finally
sorted by their journey reference.

#### ByServiceNumber

`sort.Sort(ByServiceNumber(<[]Departure>))`

Sorts departures by their service number; initially by their prefix
(services with no prefix are positioned before those with a prefix), then by
ascending digits (e.g. service `12` is positioned before service `123`), then
finally by the suffix (services with no suffix are positioned before those with
a suffix). In the case of equal service numbers, departures are then sorted by
their departure time (using the `expectedDepartureTime` if available, otherwise 
using the `aimedDepartureTime`), then finally by the journey reference.

Example of the sorted order:

* `12`
* `12A`
* `12B`
* `123`
* `123A`
* `123B`
* `A12`
* `A12A`
* `A12B`
* `B12`
* `B12A`
* `B12B`

## Output

A simplified format for outputting data suitable for consumption by downstream 
services; e.g. AWS API Gateway

```json
{
  "departures": [
      {
        "departureTime": "Approaching",
        "stand": "B",
        "serviceNumber": "42",
        "destination": "Minas Tirith"
      },
      {
        "departureTime": "12:34",
        "stand": "A",
        "serviceNumber": "123A",
        "destination": "Hobbiton"
      },
      {
        "departureTime": "12 mins",
        "stand": "G",
        "serviceNumber": "8",
        "destination": "Mordor"
    }
  ]
}
```

## SIRI

A set of structs representing the [SIRI Stop Monitoring](http://user47094.vs.easily.co.uk/siri/schema/1.3/examples/index.htm) 
service request and delivery standards.

Used internally to extract data from SIRI responses.
