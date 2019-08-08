# siri-proxy

A webserver application that proxies a SIRI endpoint.

The intended use case is to act as a man-in-the-middle between bus operators'
SIRI feeds and OPTIS; this will allow us to source data from the operators
directly, rather than relying on OPTIS.

## How it works
1) The client (OPTIS) makes a `SubscriptionRequest` to the SIRI-Proxy;
1) The SIRI-Proxy stores the subscription request details;
1) The SIRI-Proxy modifies `SubscriptionRequest`, changing the `ConsumerAddress`
  value to the URL of the SIRI-Proxy;
1) The SIRI-Proxy sends the modified `SubscriptionRequest` to the bus operator's
   server.
1) The bus operator's server responds with a `SubscriptionResponse`, which is
   sent unmodified to the client.
1) The bus operator's server publishes `ServiceDelivery` data to the SIRI-Proxy, 
   which is then sent unmodified to the client. 

As we are now receiving this data first ourselves rather than it going directly 
to OPTIS, we can include functionality which sends the data elsewhere, 
e.g. publish it to an SNS topic, where other services could consume it.

## Dockerfile
The provided [Dockerfile](Dockerfile) will run the SIRI-Proxy.

### Build the Docker container

```shell script
docker build \
    --build-arg GITHUB_ACCESS_TOKEN=X \
    -t siri-proxy-optis . 
```

_Note: The `GITHUB_ACCESS_TOKEN` must give access to the 
**TfGMEnterprise/departures-service** repository on GitHub_

### Run the built container

```shell script
docker run \
    -p 8080:8080 \
    --env SIRI_PROXY_SERVER_PORT=8080 \
    --env SIRI_PROXY_SERVER_URL=http://example.com \
    --env SIRI_PROXY_TARGET_URL=http://example.com \
    --env SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL=PT5M \
    --env HTTP_CLIENT_TIMEOUT=10 \
    --name siri-proxy-optis \
    siri-proxy-optis 
```

## Environment variables
* **SIRI_PROXY_SERVER_URL**: The public URL for accessing this proxy server. 
  This is the address to which clients will send requests; its value is used
  to replace the `ConsumerAddress` value in SIRI `SubscriptionRequest` payloads
* **SIRI_PROXY_SERVER_PORT**: The port on which the proxy server should run.
  _Defaults to `8080`._
* **SIRI_TARGET_SERVER_URL**: The URL of the target server (i.e. the SIRI feed
  provided by the bus operator).
* **SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL**: The default period of time
  for sending `HeartbeatNotification` requests to clients. _Defaults to `PT5M` 
  (every 5 minutes)._
* **HTTP_CLIENT_TIMEOUT**: The number of seconds to allow for requests before
  timing out. _Defaults to `10` seconds_

## Testing
In order to test the SIRI-Proxy, you will need:
* A SIRI feed (e.g. from a bus operator);
* A configured SIRI-Proxy running at a URL that can be accessed by the SIRI feed 
  and the test client server; 
* A client server that accepts `POST` requests.

You can post a `SubscriptionRequest` to the SIRI-Proxy (e.g. using Postman).

The example below is for a `StopMonitoringSubscriptionRequest` (other types of
subscription are available). Replace all the values in double-braces as
appropriate before submitting the request:

_Note: The `ConsumerAddress` in your `SubscriptionRequest` is the address of
your test client server._

```xml
<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri ">
    <SubscriptionRequest>
        <RequestTimestamp>{{RequestTimestamp}}</RequestTimestamp>
        <RequestorRef>{{RequestorRef}}</RequestorRef>
        <ConsumerAddress>{{ConsumerAddress}}</ConsumerAddress>
        <StopMonitoringSubscriptionRequest>
            <SubscriberRef>{{SubscriberRef}}</SubscriberRef>
            <SubscriptionIdentifier>{{SubscriptionIdentifier}}</SubscriptionIdentifier>
            <InitialTerminationTime>{{InitialTerminationTime}}</InitialTerminationTime>
            <StopMonitoringRequest version="1.3">
                <RequestTimestamp>{{RequestTimestamp}}</RequestTimestamp>
                <MonitoringRef>{{MonitoringRef}}</MonitoringRef>
                <PreviewInterval>{{PreviewInterval}}</PreviewInterval>
                <MaximumStopVisits>{{MaximumStopVisits}}</MaximumStopVisits>
            </StopMonitoringRequest>
            <ChangeBeforeUpdates>{{ChangeBeforeUpdates}}</ChangeBeforeUpdates>
    </StopMonitoringSubscriptionRequest>
</SubscriptionRequest>
</Siri>
```

If everything has worked OK, you should receive a `SubscriptionResponse`,
similar to the below:

```xml
<Siri xsi:schemaLocation="http://www.siri.org.uk/siri " version="1.3"
	xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
	<SubscriptionResponse>
		<ResponseTimestamp>2019-07-31T16:35:52Z</ResponseTimestamp>
		<ResponderRef />
		<ResponseStatus>
			<ResponseTimestamp>2019-07-31T16:35:52Z</ResponseTimestamp>
			<SubscriberRef>{{SubscriberRef}}</SubscriberRef>
			<SubscriptionRef>{{SubscriptionIdentifier}}</SubscriptionRef>
			<Status>true</Status>
		</ResponseStatus>
	</SubscriptionResponse>
</Siri>
```

You should then start to see `ServiceDelivery` and `HeartbeatNotification`
payloads being posted to your client server.

An example `ServiceDelivery` payload:

```xml
<?xml version="1.0" encoding="utf-8"?>
<Siri xsi:schemaLocation="http://www.siri.org.uk/siri " version="1.3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
    <ServiceDelivery>
        <ResponseTimestamp>2019-07-31T17:18:40Z</ResponseTimestamp>
        <ProducerRef>{{ProducerRef}}</ProducerRef>
        <Status>true</Status>
        <StopMonitoringDelivery version="1.3">
            <ResponseTimestamp>2019-07-31T17:18:40Z</ResponseTimestamp>
            <SubscriberRef>{{SubscriberRef}}</SubscriberRef>
            <SubscriptionRef>{{SubscriptionIdentifier}}</SubscriptionRef>
            <Status>true</Status>
            <MonitoredStopVisit>
                <RecordedAtTime>2019-07-31T18:18:40+01:00</RecordedAtTime>
                <ItemIdentifier>7560203-1</ItemIdentifier>
                <MonitoringRef>1800BNIN0C1</MonitoringRef>
                <MonitoredVehicleJourney>
                    <LineRef>534</LineRef>
                    <DirectionRef>outbound</DirectionRef>
                    <FramedVehicleJourneyRef>
                        <DataFrameRef>2019-07-31</DataFrameRef>
                        <DatedVehicleJourneyRef>1093</DatedVehicleJourneyRef>
                    </FramedVehicleJourneyRef>
                    <JourneyPatternRef>1054579</JourneyPatternRef>
                    <Bearing>0</Bearing>
                    <PublishedLineName></PublishedLineName>
                    <DirectionName>outbound</DirectionName>
                    <OperatorRef>ANW</OperatorRef>
                    <OriginRef>1800BNIN0C1</OriginRef>
                    <OriginName>Bolton Interchange</OriginName>
                    <DestinationRef>1800WA12481</DestinationRef>
                    <DestinationName>Oldhams Estate Terminus</DestinationName>
                    <VehicleJourneyName>1093</VehicleJourneyName>
                    <OriginAimedDepartureTime>2019-07-31T18:43:00+01:00</OriginAimedDepartureTime>
                    <DestinationAimedArrivalTime>2019-07-31T19:03:00+01:00</DestinationAimedArrivalTime>
                    <Monitored>true</Monitored>
                    <BlockRef>1032</BlockRef>
                    <VehicleRef>ANW-2948</VehicleRef>
                    <VehicleLocation>
                        <Longitude>371789</Longitude>
                        <Latitude>408852</Latitude>
                    </VehicleLocation>
                    <MonitoredCall>
                        <StopPointRef>1800BNIN0C1</StopPointRef>
                        <Order>1</Order>
                        <StopPointName>Bolton Interchange</StopPointName>
                        <VehicleAtStop>false</VehicleAtStop>
                        <TimingPoint>false</TimingPoint>
                        <ArrivalPlatformName>Bolton Interchange</ArrivalPlatformName>
                        <DeparturePlatformName>Bolton Interchange</DeparturePlatformName>
                        <AimedDepartureTime>2019-07-31T18:43:00+01:00</AimedDepartureTime>
                        <ExpectedDepartureTime>2019-07-31T18:43:00+01:00</ExpectedDepartureTime>
                        <DepartureStatus />
                    </MonitoredCall>
                </MonitoredVehicleJourney>
            </MonitoredStopVisit>
        </StopMonitoringDelivery>
    </ServiceDelivery>
</Siri>
```

An example `HeartbeatNotification` payload:

```xml
<?xml version="1.0" encoding="utf-8"?>
<Siri version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri " xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
    <HeartbeatNotification>
        <RequestTimestamp>2019-07-31T17:12:54Z</RequestTimestamp>
        <ProducerRef>{{ProducerRef}}</ProducerRef>
        <MessageIdentifier>{{SubscriberRef}}</MessageIdentifier>
        <Status>true</Status>
        <ServiceStartedTime>2019-07-28T01:37:10Z</ServiceStartedTime>
    </HeartbeatNotification>
</Siri>
```
