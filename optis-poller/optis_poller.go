package main

import (
	"encoding/json"
	"github.com/ChannelMeter/iso8601duration"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	optis_client "github.com/TfGMEnterprise/departures-service/optis-client"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// BusStation is the location we are requesting data for
type BusStation struct {
	Atcocode string `json:"atcocode"`
}

type OptisPoller struct {
	Logger                 *dlog.Logger
	OptisClient            optis_client.OptisClientInterface
	OptisMaximumStopVisits int
	OptisPreviewInterval   duration.Duration
	OptisRequestorRef      string
	SNSClient              snsiface.SNSAPI
	SNSTopicARN            *string
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("optis-poller: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	optisUrl, exists := os.LookupEnv("OPTIS_STOP_MONITORING_REQUEST_URL")
	if !exists || optisUrl == "" {
		logger.Fatal("OPTIS_STOP_MONITORING_REQUEST_URL not set in environment")
	}

	optisAPIKey, exists := os.LookupEnv("OPTIS_API_KEY")
	if !exists || optisAPIKey == "" {
		logger.Fatal("OPTIS_API_KEY not set in environment")
	}

	optisRequestorRef, exists := os.LookupEnv("OPTIS_REQUESTOR_REF")
	if !exists || optisRequestorRef == "" {
		logger.Fatal("OPTIS_REQUESTOR_REF not set in environment")
	}

	previewIntervalString, exists := os.LookupEnv("OPTIS_PREVIEW_INTERVAL")
	if !exists || previewIntervalString == "" {
		logger.Fatal("OPTIS_PREVIEW_INTERVAL not set in environment")
	}
	maximumStopVisits, exists := os.LookupEnv("OPTIS_MAXIMUM_STOP_VISITS")
	if !exists || maximumStopVisits == "" {
		logger.Fatal("OPTIS_MAXIMUM_STOP_VISITS not set in environment")
	}

	optisTimeoutStr, exists := os.LookupEnv("OPTIS_TIMEOUT")
	if !exists || optisTimeoutStr == "" {
		optisTimeoutStr = "30"
	}

	optisTimeout, err := strconv.Atoi(optisTimeoutStr)
	if err != nil {
		logger.Fatal("OPTIS_TIMEOUT value is invalid")
	}

	if optisTimeout <= 0 {
		logger.Fatal("OPTIS_TIMEOUT value must be greater than 0")
	}

	snsTopicURN, exists := os.LookupEnv("AWS_SNS_TOPIC_ARN")
	if !exists || snsTopicURN == "" {
		logger.Fatal("AWS_SNS_TOPIC_ARN not set in environment")
	}

	optisPreviewInterval, err := duration.FromString(previewIntervalString)
	if err != nil {
		logger.Fatal(errors.Wrapf(err, "OPTIS_PREVIEW_INTERVAL value `%s` is not a valid ISO8601 duration", previewIntervalString))
		return
	}

	optisMaximumStopVisits, err := strconv.Atoi(maximumStopVisits)
	if err != nil {
		logger.Fatal(errors.Wrapf(err, "OPTIS_MAXIMUM_STOP_VISITS value `%s` is not valid", maximumStopVisits))
		return
	}

	optisClient := optis_client.OptisClient{
		Client: &http.Client{
			Timeout: time.Second * time.Duration(optisTimeout),
		},
		Logger:      logger,
		OptisURL:    optisUrl,
		OptisAPIKey: optisAPIKey,
	}

	sess := session.Must(session.NewSession())

	snsClient := *sns.New(sess)

	op := OptisPoller{
		Logger:                 logger,
		OptisClient:            &optisClient,
		OptisMaximumStopVisits: optisMaximumStopVisits,
		OptisPreviewInterval:   *optisPreviewInterval,
		OptisRequestorRef:      optisRequestorRef,
		SNSClient:              &snsClient,
		SNSTopicARN:            &snsTopicURN,
	}

	lambda.Start(op.Handler)
}

func (op *OptisPoller) Handler(busStation BusStation) error {
	op.Logger.Debug("Handler")

	siriRequest := op.createSiriRequest(busStation.Atcocode)
	siriResponse, httpStatus, err := op.OptisClient.Request(siriRequest)
	if err != nil {
		return errors.Wrapf(err, "request to OPTIS failed with status `%d`", httpStatus)
	}

	if err := op.checkHasDepartures(siriResponse); err != nil {
		return errors.Wrap(err, "request to OPTIS failed")
	}

	op.filter(siriResponse)

	departures := op.transform(siriResponse)

	departuresJSON, err := json.Marshal(&departures)
	message := aws.String(string(departuresJSON))
	if err != nil {
		return errors.Wrap(err, "cannot marshal JSON from departure")
	}

	if _, err := op.SNSClient.Publish(&sns.PublishInput{
		Message:  message,
		TopicArn: op.SNSTopicARN,
	}); err != nil {
		return errors.Wrapf(err, "cannot publish message to SNS topic `%s`", *op.SNSTopicARN)
	}

	return nil
}

func (op *OptisPoller) createSiriRequest(monitoringRef string) string {
	op.Logger.Debugf("createSiriRequest for `%s`", monitoringRef)
	requestTimestamp := time.Now()
	return `<?xml version="1.0" encoding="UTF-8" ?>
<Siri xmlns="http://www.siri.org.uk/siri" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version = "1.3" xsi:schemaLocation = "http://www.siri.org.uk/siri">
    <ServiceRequest>
        <RequestTimestamp>` + requestTimestamp.Format(time.RFC3339) + `</RequestTimestamp>
        <RequestorRef>` + op.OptisRequestorRef + `</RequestorRef>
        <StopMonitoringRequest version="1.3">
            <RequestTimestamp>` + requestTimestamp.Format(time.RFC3339) + `</RequestTimestamp>
            <MonitoringRef>` + monitoringRef + `</MonitoringRef>
            <PreviewInterval>` + op.OptisPreviewInterval.String() + `</PreviewInterval>
            <MaximumStopVisits>` + strconv.Itoa(op.OptisMaximumStopVisits) + `</MaximumStopVisits>
        </StopMonitoringRequest>
    </ServiceRequest>
</Siri>`
}

func (op *OptisPoller) checkHasDepartures(siriResponse *model.Siri) error {
	op.Logger.Debug("checkHasDepartures")
	if !siriResponse.ServiceDelivery.StopMonitoringDelivery.Status {
		return errors.New("request failed; no stop monitoring delivery")
	}

	return nil
}

func (op *OptisPoller) filter(siri *model.Siri) {
	op.Logger.Debug("filter")
	i := 0
	initialLen := len(siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit)
	op.Logger.Debugf("filter - %d records to filter", initialLen)
	for _, monitoredStopVisit := range siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit {
		if op.hasDepartureTime(&monitoredStopVisit.MonitoredVehicleJourney.MonitoredCall) &&
			!op.erroneousRecord(&monitoredStopVisit.MonitoredVehicleJourney) &&
			!op.cancelledJourney(&monitoredStopVisit.MonitoredVehicleJourney) {
			siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit[i] = monitoredStopVisit
			i++
			op.Logger.Debugf("include JourneyRef %s", op.getMonitoredJourneyIdentity(&monitoredStopVisit.MonitoredVehicleJourney))
			continue
		}
		op.Logger.Debugf("exclude JourneyRef %s", op.getMonitoredJourneyIdentity(&monitoredStopVisit.MonitoredVehicleJourney))
	}
	siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit = siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit[:i]
	op.Logger.Debugf("filter - %d records remain; %d records filtered", i, initialLen-i)
}

func (op *OptisPoller) getMonitoredJourneyIdentity(monitoredVehicleJourney *model.MonitoredVehicleJourney) string {
	return strings.Join([]string{
		monitoredVehicleJourney.LineRef,
		monitoredVehicleJourney.DirectionRef,
		monitoredVehicleJourney.FramedVehicleJourneyRef.DataFrameRef,
		monitoredVehicleJourney.FramedVehicleJourneyRef.DatedVehicleJourneyRef,
	}, "_")
}

func (op *OptisPoller) hasDepartureTime(call *model.MonitoredCall) bool {
	op.Logger.Debug("hasDepartureTime")
	zeroTime := time.Time{}
	op.Logger.Debugf("ExpectedDepartureTime is %s; AimedDepartureTime is %s; ZeroTime is %s", call.ExpectedDepartureTime.Format(time.RFC3339), call.AimedDepartureTime.Format(time.RFC3339), zeroTime.Format(time.RFC3339))
	return call.ExpectedDepartureTime != zeroTime || call.AimedDepartureTime != zeroTime
}

func (op *OptisPoller) erroneousRecord(monitoredVehicleJourney *model.MonitoredVehicleJourney) bool {
	op.Logger.Debug("notErroneousRecord")
	erroneous := monitoredVehicleJourney.MonitoredCall.AimedDepartureTime.Before(monitoredVehicleJourney.OriginAimedDepartureTime)
	if erroneous {
		op.Logger.Printf("erroneous record: AimedDepartureTime %s is before OriginAimedDepartureTime %s", monitoredVehicleJourney.MonitoredCall.AimedDepartureTime, monitoredVehicleJourney.OriginAimedDepartureTime)
	}
	return erroneous
}

func (op *OptisPoller) cancelledJourney(monitoredVehicleJourney *model.MonitoredVehicleJourney) bool {
	op.Logger.Debug("cancelledJourney")
	cancelled := monitoredVehicleJourney.MonitoredCall.DepartureStatus == "cancelled"
	if cancelled {
		op.Logger.Printf("cancelled journey at %s: %s scheduled to depart at %s", monitoredVehicleJourney.MonitoredCall.StopPointRef, monitoredVehicleJourney.LineRef, monitoredVehicleJourney.MonitoredCall.AimedDepartureTime)
	}
	return cancelled
}

func (op *OptisPoller) isZeroTime(ts time.Time) bool {
	op.Logger.Debugf("isZeroTime `%s`", ts.Format(time.RFC3339))
	return ts == time.Time{}
}

func (op *OptisPoller) transform(siri *model.Siri) model.Internal {
	op.Logger.Debug("transform")
	departures := model.Internal{}

	for _, monitoredStopVisit := range siri.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit {
		departure := model.Departure{
			RecordedAtTime:      monitoredStopVisit.RecordedAtTime.Format(time.RFC3339),
			JourneyType:         model.Bus,
			JourneyRef:          op.getMonitoredJourneyIdentity(&monitoredStopVisit.MonitoredVehicleJourney),
			AimedDepartureTime:  monitoredStopVisit.MonitoredVehicleJourney.MonitoredCall.AimedDepartureTime.Format(time.RFC3339),
			LocationAtcocode:    monitoredStopVisit.MonitoredVehicleJourney.MonitoredCall.StopPointRef,
			DestinationAtcocode: monitoredStopVisit.MonitoredVehicleJourney.DestinationRef,
			Destination:         monitoredStopVisit.MonitoredVehicleJourney.DestinationName,
			ServiceNumber:       monitoredStopVisit.MonitoredVehicleJourney.LineRef,
			OperatorCode:        monitoredStopVisit.Extensions.NationalOperatorCode,
		}

		if !op.isZeroTime(monitoredStopVisit.MonitoredVehicleJourney.MonitoredCall.ExpectedDepartureTime) {
			expectedDepartureTime := monitoredStopVisit.MonitoredVehicleJourney.MonitoredCall.ExpectedDepartureTime.Format(time.RFC3339)
			departure.ExpectedDepartureTime = &expectedDepartureTime
		}

		if stand := departure.GetStand(); stand != nil {
			departure.Stand = stand
		}

		departures.Departures = append(departures.Departures, departure)
	}

	return departures
}
