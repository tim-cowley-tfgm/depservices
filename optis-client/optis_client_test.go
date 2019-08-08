package optis_client

import (
	"encoding/xml"
	"fmt"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	now = time.Now()

	SuccessfulStopMonitoringResponse = `<?xml version="1.0" encoding="utf-8"?>
<Siri xsi:schemaLocation="http://www.siri.org.uk/siri " version="1.3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
    <ServiceDelivery>
        <ResponseTimestamp>` + now.Format(time.RFC3339) + `</ResponseTimestamp>
        <ProducerRef>CloudAmber</ProducerRef>
        <Status>true</Status>
        <MoreData>true</MoreData>
        <StopMonitoringDelivery version="1.3">
            <ResponseTimestamp>` + now.Format(time.RFC3339) + `</ResponseTimestamp>
            <SubscriptionFilterRef />
            <Status>true</Status>
            <ValidUntil>` + now.Format(time.RFC3339) + `</ValidUntil>
            <MonitoredStopVisit>
                <RecordedAtTime>` + now.Format(time.RFC3339) + `</RecordedAtTime>
                <MonitoringRef>1800BNIN0A1</MonitoringRef>
                <MonitoredVehicleJourney>
                    <LineRef>1</LineRef>
                    <DirectionRef>inbound</DirectionRef>
                    <FramedVehicleJourneyRef>
                        <DataFrameRef>2019-04-26</DataFrameRef>
                        <DatedVehicleJourneyRef>1078</DatedVehicleJourneyRef>
                    </FramedVehicleJourneyRef>
                    <JourneyPatternRef>424016</JourneyPatternRef>
                    <DirectionName>inbound</DirectionName>
                    <OperatorRef>ANW</OperatorRef>
                    <VehicleFeatureRef>WheelchairAccessible</VehicleFeatureRef>
                    <OriginRef>1800WA12481</OriginRef>
                    <OriginName>Turning Circle</OriginName>
                    <DestinationRef>1800NE43431</DestinationRef>
                    <Destination>Hobbiton</Destination>
                    <VehicleJourneyName>1078</VehicleJourneyName>
                    <OriginAimedDepartureTime>2019-04-26T15:50:00+01:00</OriginAimedDepartureTime>
                    <DestinationAimedArrivalTime>2019-04-26T16:10:00+01:00</DestinationAimedArrivalTime>
                    <Monitored>false</Monitored>
                    <VehicleLocation>
                        <Longitude>-2.432795</Longitude>
                        <Latitude>53.59906</Latitude>
                    </VehicleLocation>
                    <BlockRef>1031</BlockRef>
                    <VehicleRef>ANW-2527</VehicleRef>
                    <MonitoredCall>
                        <StopPointRef>1800BNIN0A1</StopPointRef>
                        <Order>20</Order>
                        <StopPointName>Bolton Interchange</StopPointName>
                        <VehicleAtStop>false</VehicleAtStop>
                        <TimingPoint>false</TimingPoint>
                        <DestinationDisplay />
                        <AimedArrivalTime>` + test_helpers.AdjustTime(now, "3m8s").Format(time.RFC3339) + `</AimedArrivalTime>
                        <ExpectedArrivalTime>` + test_helpers.AdjustTime(now, "59s").Format(time.RFC3339) + `</ExpectedArrivalTime>
                        <ArrivalStatus>PickupAndSetDown</ArrivalStatus>
                        <ArrivalPlatformName>Bolton Interchange</ArrivalPlatformName>
                        <ArrivalBoardingActivity>alighting</ArrivalBoardingActivity>
                        <DeparturePlatformName>Bolton Interchange</DeparturePlatformName>
                        <AimedDepartureTime>` + test_helpers.AdjustTime(now, "3m8s").Format(time.RFC3339) + `</AimedDepartureTime>
                        <ExpectedDepartureTime>` + test_helpers.AdjustTime(now, "59s").Format(time.RFC3339) + `</ExpectedDepartureTime>
                    </MonitoredCall>
                </MonitoredVehicleJourney>
                <Extensions>
                    <NationalOperatorCode xmlns="">ANWE</NationalOperatorCode>
                </Extensions>
            </MonitoredStopVisit>
            <Note />
        </StopMonitoringDelivery>
    </ServiceDelivery>
</Siri>`
	FailedStopMonitoringResponse = `<?xml version="1.0" encoding="utf-8"?>
<Siri xsi:schemaLocation="http://www.siri.org.uk/siri " version="1.3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
    <ServiceDelivery>
        <ResponseTimestamp>` + time.Now().Format(time.RFC3339) + `</ResponseTimestamp>
        <ProducerRef>CloudAmber</ProducerRef>
        <Status>false</Status>
        <MoreData>false</MoreData>
    </ServiceDelivery>
</Siri>`
	ErrorResponse = `<Siri xsi:schemaLocation="http://www.siri.org.uk/siri " version="1.3" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns="http://www.siri.org.uk/siri">
    <ServiceDelivery>
        <Status>false</Status>
        <ErrorCondition>
            <Description>Requestorref not subscribed to $service.</Description>
        </ErrorCondition>
    </ServiceDelivery>
</Siri>`
	InvalidAPIKeyResponse = `Client can not access the desired service -> SIRI-SM -> Access denied`
	MissingAPIKeyResponse = `Client can not access the desired service -> SIRI-SM -> Missing apiKey`
)

func TestOptisClient_Handler(t *testing.T) {
	optisAPIKey := "abc123"
	atcocode := "1800BNIN"
	requestorRef := "OPTIS_TEST"
	maximumStopVisits := 50

	createOptisStub := func() *httptest.Server {
		t.Helper()
		var siriResponse string
		var statusCode int

		optisHandlerFunc := func(w http.ResponseWriter, r *http.Request) {
			t.Helper()
			APIKey := strings.Join(r.URL.Query()["apiKey"], "")

			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("%s\n", err.Error())
			}

			requestBody := new(model.Siri)

			if err := xml.Unmarshal(body, &requestBody); err != nil {
				t.Fatalf("%s\n", err.Error())
			}

			switch true {
			case APIKey == "":
				siriResponse = MissingAPIKeyResponse
				statusCode = http.StatusUnauthorized
			case APIKey != optisAPIKey:
				siriResponse = InvalidAPIKeyResponse
				statusCode = http.StatusUnauthorized
			case requestBody.ServiceRequest.RequestorRef != requestorRef:
				siriResponse = ErrorResponse
				statusCode = http.StatusOK
			case requestBody.ServiceRequest.StopMonitoringRequest.MaximumStopVisits != maximumStopVisits:
				siriResponse = FailedStopMonitoringResponse
				statusCode = http.StatusOK
			default:
				siriResponse = SuccessfulStopMonitoringResponse
				statusCode = http.StatusOK
			}

			w.WriteHeader(statusCode)
			// For some reason OPTIS sets the content-type as "text/html" rather than something more appropriate;
			// e.g. "application/xml"
			w.Header().Set("Content-Type", "text/html")
			if _, err := fmt.Fprintf(w, siriResponse); err != nil {
				t.Fatalf("%s\n", err.Error())
			}
		}

		return httptest.NewServer(http.HandlerFunc(optisHandlerFunc))
	}

	t.Run("happy path", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: optisAPIKey,
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits>` + strconv.Itoa(maximumStopVisits) + `</MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		siriResponse, statusCode, err := o.Request(siriRequest)
		if err != nil {
			t.Errorf("%s\n", err.Error())
		}

		if statusCode != http.StatusOK {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusOK, statusCode)
		}

		expectedSiriResponse := new(model.Siri)

		if err := xml.Unmarshal([]byte(SuccessfulStopMonitoringResponse), &expectedSiriResponse); err != nil {
			t.Fatalf("%s\n", err.Error())
		}

		if reflect.DeepEqual(siriResponse, expectedSiriResponse) == false {
			t.Errorf("Got unexpected SIRI response; got:\n%#v\nwant:\n%#v", siriResponse, expectedSiriResponse)
		}
	})

	t.Run("no API key", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: "",
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits>` + strconv.Itoa(maximumStopVisits) + `</MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusUnauthorized {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("invalid API key", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: "invalid",
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits>` + strconv.Itoa(maximumStopVisits) + `</MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusUnauthorized {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusUnauthorized, statusCode)
		}
	})

	t.Run("invalid requestorRef", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: optisAPIKey,
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>invalid</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits>` + strconv.Itoa(maximumStopVisits) + `</MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusBadRequest {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusBadRequest, statusCode)
		}
	})

	t.Run("invalid value in request body", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: optisAPIKey,
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits></MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusBadRequest {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusBadRequest, statusCode)
		}
	})

	t.Run("no response from OPTIS", func(t *testing.T) {
		optisStub := createOptisStub()
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    "http://foo.bar",
			OptisAPIKey: optisAPIKey,
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits></MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusGatewayTimeout {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusGatewayTimeout, statusCode)
		}
	})

	t.Run("error response from OPTIS", func(t *testing.T) {
		optisStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Helper()

			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer optisStub.Close()

		o := OptisClient{
			Client: optisStub.Client(),
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			OptisURL:    optisStub.URL,
			OptisAPIKey: optisAPIKey,
		}

		siriRequest := `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.3" xsi:schemaLocation="http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + requestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + time.Now().Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + atcocode + `</MonitoringRef>
            <MaximumStopVisits></MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`

		_, statusCode, err := o.Request(siriRequest)
		if err == nil {
			t.Error("Expected an error; no error returned")
		}

		if statusCode != http.StatusBadGateway {
			t.Errorf("Want HTTP status code: %d; got: %d\n", http.StatusBadGateway, statusCode)
		}
	})
}
