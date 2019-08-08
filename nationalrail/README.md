# nationalrail

Contains the WSDL-based structs for accessing the National Rail Enquiries (NRE)
[Live Departure Boards Web Service](http://lite.realtime.nationalrail.co.uk/openldbws/) 
feed.

Also contains the GetAtcoCode function, which expects a three-character CRS
(Computer Reservation System) code for a station in Greater Manchester and 
returns the NaPTAN ATCO Code for that location.

## WSDL

The feed is a request/response SOAP API; SOAP isn't the easiest thing in the
world to work with! However, we are able to generate most of what we need using
 [gowsdl](https://github.com/hooklift/gowsdl):

```
gowsdl -o generated.go -p nationalrail https://realtime.nationalrail.co.uk/ldbws/rtti_2017-10-01_ldb.wsdl
```

We need to modify the remove the duplicate types that are created, as
well add adding the SOAP Header and AccessToken types:

```go
type SOAPHeader struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`

	Header interface{} `xml:",omitempty"`
}

type AccessToken struct {
	XMLName xml.Name `xml:"http://thalesgroup.com/RTTI/2010-11-01/ldb/commontypes AccessToken"`

	TokenValue string `xml:"TokenValue"`
}
```

We also add field flags for JSON formatting.

Once done, we can then import the `nationalrail` package for use elsewhere.

## GetAtcoCode

The GetAtcoCode function simply references a map of static data, which is
derived from the `RailReferences.csv` file in the NaPTANcsv dataset.

This function supports us in providing a consistent query pattern at the
presentation level, i.e. we use the ATCO code for all locations, rather than
ATCO codes for buses, CRS codes for rail, etc.

Given that our scope is Greater Manchester and the last railway station to open
in Greater Manchester was Horwich Parkway in 1999 and there currently aren't any
new railway stations planned or under construction, it was not considered
worthwhile to set up a database/cache to store the relationship between CRS
codes and ATCO codes.

If that situation should change, the [rail-references](../rail-references/README.md)
Lambda function will provide the necessary functionality to store the 
relationship in a Redis cache.
