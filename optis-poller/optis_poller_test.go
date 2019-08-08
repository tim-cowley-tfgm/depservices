package main

import (
	"encoding/xml"
	"errors"
	"github.com/ChannelMeter/iso8601duration"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/fortytw2/leaktest"
	"io/ioutil"
	"net/http"
	"reflect"
	"testing"
	"time"
)

type MockOptisClient struct {
	OptisURL    string
	OptisAPIKey string
}

type MockSNSClient struct {
	snsiface.SNSAPI
	MockSNSInterface
	PublishCallCount   int
	PublishExpectation sns.PublishInput
	Output             sns.PublishOutput
	T                  *testing.T
}

type MockSNSInterface interface {
	SetPublishExpectation(string)
}

func (ms *MockSNSClient) SetPublishExpectation(expectation sns.PublishInput) {
	ms.PublishExpectation = expectation
}

func (ms *MockSNSClient) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	ms.T.Helper()

	if !reflect.DeepEqual(input, &ms.PublishExpectation) {
		ms.T.Errorf("Publish to SNS:\ngot:\n%#v\nwant:\n%#v\n", input, &ms.PublishExpectation)
	}

	ms.PublishCallCount = ms.PublishCallCount + 1

	return &ms.Output, nil
}

const (
	optisStopMonitoringRequestUrl = "http://foo.bar/"
	optisAPIKey                   = "abc123"
	busStationAtcocode            = "1800BNIN"
	requestorRef                  = "OPTIS_TEST"
	previewInterval               = "PT1H30M"
	maximumStopVisits             = 50
	snsTopicArn                   = "arn:aws:sns:mars-north-8:123456789012:optis-departures"
)

var (
	now = time.Now()

	previewIntervalDuration, _ = duration.FromString(previewInterval)

	ErrorConditionSiriResponse = model.Siri{
		ServiceDelivery: model.ServiceDelivery{
			Status: false,
			ErrorCondition: model.ErrorCondition{
				Description: "Requestorref not subscribed to StopMonitoring single shot.",
			},
		},
	}
	InvalidAtcoCodeSiriResponse = model.Siri{
		ServiceDelivery: model.ServiceDelivery{
			Status:   true,
			MoreData: false,
			StopMonitoringDelivery: model.StopMonitoringDelivery{
				Status: true,
			},
		},
	}
	StatusFalseSiriResponse = model.Siri{
		ServiceDelivery: model.ServiceDelivery{
			Status: false,
		},
	}
	HappyBusStationSiriResponse = model.Siri{
		ServiceDelivery: model.ServiceDelivery{
			Status: true,
			StopMonitoringDelivery: model.StopMonitoringDelivery{
				Status: true,
				MonitoredStopVisit: []model.MonitoredStopVisit{
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0A1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "1",
							DirectionRef: "inbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-09",
								DatedVehicleJourneyRef: "0001",
							},
							DestinationRef:              "1800HN00011",
							DestinationName:             "Hobbiton",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:          "1800BNIN0A1",
								AimedDepartureTime:    test_helpers.AdjustTime(now, "3m8s"),
								ExpectedDepartureTime: test_helpers.AdjustTime(now, "59s"),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0B1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "2",
							DirectionRef: "outbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-09",
								DatedVehicleJourneyRef: "0002",
							},
							DestinationRef:              "1800MD00011",
							DestinationName:             "Mordor",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:       "1800BNIN0B1",
								AimedDepartureTime: test_helpers.AdjustTime(now, "1m10s"),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0C1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "3",
							DirectionRef: "outbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-09",
								DatedVehicleJourneyRef: "0003",
							},
							DestinationRef:              "1800MT00011",
							DestinationName:             "Minas Tirith",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:          "1800BNIN0C1",
								AimedDepartureTime:    test_helpers.AdjustTime(now, "3m8s"),
								ExpectedDepartureTime: test_helpers.AdjustTime(now, "1m1s"),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0D1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "4",
							DirectionRef: "inbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-09",
								DatedVehicleJourneyRef: "0004",
							},
							DestinationRef:              "1800BR00011",
							DestinationName:             "Bree",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:          "1800BNIN0D1",
								AimedDepartureTime:    test_helpers.AdjustTime(now, "3m8s"),
								ExpectedDepartureTime: test_helpers.AdjustTime(now, "2m59s"),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0E1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "5",
							DirectionRef: "inbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-09",
								DatedVehicleJourneyRef: "0005",
							},
							DestinationRef:              "1800BR00021",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:        "1800BNIN0E1",
								AimedArrivalTime:    test_helpers.AdjustTime(now, "3m8s"),
								ExpectedArrivalTime: test_helpers.AdjustTime(now, "3m8s"),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0F1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "6",
							DirectionRef: "inbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-10",
								DatedVehicleJourneyRef: "0006",
							},
							DestinationRef:              "1800BR00021",
							OriginAimedDepartureTime:    time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Add(10 * time.Hour),
							DestinationAimedArrivalTime: time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Add(12 * time.Hour),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:       "1800BNIN0F1",
								AimedArrivalTime:   time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour).Add(12 * time.Hour),
								AimedDepartureTime: time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour),
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
					{
						RecordedAtTime: time.Now(),
						MonitoringRef:  "1800BNIN0G1",
						MonitoredVehicleJourney: model.MonitoredVehicleJourney{
							LineRef:      "7",
							DirectionRef: "inbound",
							FramedVehicleJourneyRef: model.FramedVehicleJourneyRef{
								DataFrameRef:           "2019-05-10",
								DatedVehicleJourneyRef: "0007",
							},
							DestinationRef:              "1800BR00021",
							OriginAimedDepartureTime:    test_helpers.AdjustTime(now, "-20m"),
							DestinationAimedArrivalTime: test_helpers.AdjustTime(now, "1h30m"),
							MonitoredCall: model.MonitoredCall{
								StopPointRef:       "1800BNIN0B1",
								AimedDepartureTime: test_helpers.AdjustTime(now, "1m10s"),
								DepartureStatus:    "cancelled",
							},
						},
						Extensions: model.Extensions{
							NationalOperatorCode: "ANWE",
						},
					},
				},
			},
		},
	}
)

func (mockOptisClient *MockOptisClient) Request(siriRequest string) (*model.Siri, int, error) {
	siriRequestXML := new(model.Siri)
	if err := xml.Unmarshal([]byte(siriRequest), &siriRequestXML); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	switch true {
	case siriRequestXML.ServiceRequest.RequestorRef != requestorRef:
		return &ErrorConditionSiriResponse, http.StatusBadRequest, errors.New("Requestorref not subscribed to StopMonitoring single shot.")
	case siriRequestXML.ServiceRequest.StopMonitoringRequest.MaximumStopVisits != maximumStopVisits:
		return &StatusFalseSiriResponse, http.StatusBadRequest, errors.New("OPTIS returned an error")
	case siriRequestXML.ServiceRequest.StopMonitoringRequest.MonitoringRef == busStationAtcocode:
		return &HappyBusStationSiriResponse, http.StatusOK, nil
	default:
		return &InvalidAtcoCodeSiriResponse, http.StatusOK, nil
	}
}

func TestOptisPoller_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	t.Run("happy bus station path", func(t *testing.T) {
		busStation := BusStation{
			Atcocode: busStationAtcocode,
		}

		mockedOptisClient := &MockOptisClient{
			OptisURL:    optisStopMonitoringRequestUrl,
			OptisAPIKey: optisAPIKey,
		}

		mockedSNSClient := &MockSNSClient{
			Output: sns.PublishOutput{
				MessageId: aws.String("ABC-123"),
			},
			T: t,
		}

		op := OptisPoller{
			Logger:                 logger,
			OptisClient:            mockedOptisClient,
			OptisMaximumStopVisits: maximumStopVisits,
			OptisPreviewInterval:   *previewIntervalDuration,
			OptisRequestorRef:      requestorRef,
			SNSClient:              mockedSNSClient,
			SNSTopicARN:            aws.String(snsTopicArn),
		}

		expectation := sns.PublishInput{
			Message: aws.String(`{"departures":[` +
				`{"recordedAtTime":"` + now.Format(time.RFC3339) + `","journeyType":"` + string(model.Bus) + `","journeyRef":"1_inbound_2019-05-09_0001","aimedDepartureTime":"` + test_helpers.AdjustTime(now, "3m8s").Format(time.RFC3339) + `","expectedDepartureTime":"` + test_helpers.AdjustTime(now, "59s").Format(time.RFC3339) + `","locationAtcocode":"` + busStationAtcocode + `0A1","stand":"A","destinationAtcocode":"1800HN00011","destination":"Hobbiton","serviceNumber":"1","operatorCode":"ANWE"},` +
				`{"recordedAtTime":"` + now.Format(time.RFC3339) + `","journeyType":"` + string(model.Bus) + `","journeyRef":"2_outbound_2019-05-09_0002","aimedDepartureTime":"` + test_helpers.AdjustTime(now, "1m10s").Format(time.RFC3339) + `","locationAtcocode":"` + busStationAtcocode + `0B1","stand":"B","destinationAtcocode":"1800MD00011","destination":"Mordor","serviceNumber":"2","operatorCode":"ANWE"},` +
				`{"recordedAtTime":"` + now.Format(time.RFC3339) + `","journeyType":"` + string(model.Bus) + `","journeyRef":"3_outbound_2019-05-09_0003","aimedDepartureTime":"` + test_helpers.AdjustTime(now, "3m8s").Format(time.RFC3339) + `","expectedDepartureTime":"` + test_helpers.AdjustTime(now, "1m1s").Format(time.RFC3339) + `","locationAtcocode":"` + busStationAtcocode + `0C1","stand":"C","destinationAtcocode":"1800MT00011","destination":"Minas Tirith","serviceNumber":"3","operatorCode":"ANWE"},` +
				`{"recordedAtTime":"` + now.Format(time.RFC3339) + `","journeyType":"` + string(model.Bus) + `","journeyRef":"4_inbound_2019-05-09_0004","aimedDepartureTime":"` + test_helpers.AdjustTime(now, "3m8s").Format(time.RFC3339) + `","expectedDepartureTime":"` + test_helpers.AdjustTime(now, "2m59s").Format(time.RFC3339) + `","locationAtcocode":"` + busStationAtcocode + `0D1","stand":"D","destinationAtcocode":"1800BR00011","destination":"Bree","serviceNumber":"4","operatorCode":"ANWE"}` +
				`]}`),
			TopicArn: aws.String(snsTopicArn),
		}

		mockedSNSClient.SetPublishExpectation(expectation)

		if err := op.Handler(busStation); err != nil {
			t.Error(err)
			return
		}

		if mockedSNSClient.PublishCallCount != 1 {
			t.Error("SNS Publish should have been called once.")
		}
	})

	t.Run("bad request to OPTIS because of incorrect RequestorRef", func(t *testing.T) {
		busStation := BusStation{
			Atcocode: busStationAtcocode,
		}

		mockedOptisClient := &MockOptisClient{
			OptisURL:    optisStopMonitoringRequestUrl,
			OptisAPIKey: optisAPIKey,
		}

		mockedSNSClient := &MockSNSClient{
			Output: sns.PublishOutput{
				MessageId: aws.String("ABC-123"),
			},
			T: t,
		}

		op := OptisPoller{
			Logger:                 logger,
			OptisClient:            mockedOptisClient,
			OptisMaximumStopVisits: maximumStopVisits,
			OptisPreviewInterval:   *previewIntervalDuration,
			OptisRequestorRef:      "invalid",
			SNSClient:              mockedSNSClient,
			SNSTopicARN:            aws.String(snsTopicArn),
		}

		if err := op.Handler(busStation); err == nil {
			t.Error("Error should have been returned; OPTIS Requestor Ref not set")
		}

		if mockedSNSClient.PublishCallCount != 0 {
			t.Error("Publish should not be called if the request to OPTIS fails")
		}
	})

	t.Run("missing atcocode", func(t *testing.T) {
		busStation := BusStation{}

		mockedOptisClient := &MockOptisClient{
			OptisURL:    optisStopMonitoringRequestUrl,
			OptisAPIKey: optisAPIKey,
		}

		mockedSNSClient := &MockSNSClient{
			Output: sns.PublishOutput{
				MessageId: aws.String("ABC-123"),
			},
			T: t,
		}

		op := OptisPoller{
			Logger:                 logger,
			OptisClient:            mockedOptisClient,
			OptisMaximumStopVisits: maximumStopVisits,
			OptisPreviewInterval:   *previewIntervalDuration,
			OptisRequestorRef:      requestorRef,
			SNSClient:              mockedSNSClient,
			SNSTopicARN:            aws.String(snsTopicArn),
		}

		expectation := sns.PublishInput{
			Message:  aws.String(`{"departures":null}`),
			TopicArn: aws.String(snsTopicArn),
		}

		mockedSNSClient.SetPublishExpectation(expectation)

		if err := op.Handler(busStation); err != nil {
			t.Error(err)
			return
		}

		if mockedSNSClient.PublishCallCount != 1 {
			t.Error("SNS Publish should have been called once.")
		}
	})

	t.Run("invalid atcocode", func(t *testing.T) {
		busStation := BusStation{
			Atcocode: "invalid",
		}

		mockedOptisClient := &MockOptisClient{
			OptisURL:    optisStopMonitoringRequestUrl,
			OptisAPIKey: optisAPIKey,
		}

		mockedSNSClient := &MockSNSClient{
			Output: sns.PublishOutput{
				MessageId: aws.String("ABC-123"),
			},
			T: t,
		}

		op := OptisPoller{
			Logger:                 logger,
			OptisClient:            mockedOptisClient,
			OptisMaximumStopVisits: maximumStopVisits,
			OptisPreviewInterval:   *previewIntervalDuration,
			OptisRequestorRef:      requestorRef,
			SNSClient:              mockedSNSClient,
			SNSTopicARN:            aws.String(snsTopicArn),
		}

		expectation := sns.PublishInput{
			Message:  aws.String(`{"departures":null}`),
			TopicArn: aws.String(snsTopicArn),
		}

		mockedSNSClient.SetPublishExpectation(expectation)

		if err := op.Handler(busStation); err != nil {
			t.Error(err)
			return
		}

		if mockedSNSClient.PublishCallCount != 1 {
			t.Error("SNS Publish should have been called once.")
		}
	})
}

func TestOptisPoller_hasDepatureTime(t *testing.T) {
	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	op := OptisPoller{
		Logger: logger,
	}

	t.Run("false if no expected departure time or aimed departure time", func(t *testing.T) {
		call := model.MonitoredCall{}

		got := op.hasDepartureTime(&call)

		test_helpers.AssertBoolean(t, got, false)
	})

	t.Run("true if aimed departure time", func(t *testing.T) {
		now := time.Now()
		aimedAdditionalTime, _ := time.ParseDuration("10m59s")
		aimedTime := now.Add(aimedAdditionalTime)

		call := model.MonitoredCall{
			AimedDepartureTime: aimedTime,
		}

		got := op.hasDepartureTime(&call)

		test_helpers.AssertBoolean(t, got, true)
	})

	t.Run("true if expected departure time", func(t *testing.T) {
		now := time.Now()
		aimedAdditionalTime, _ := time.ParseDuration("10m59s")
		aimedTime := now.Add(aimedAdditionalTime)

		call := model.MonitoredCall{
			ExpectedDepartureTime: aimedTime,
		}

		got := op.hasDepartureTime(&call)

		test_helpers.AssertBoolean(t, got, true)
	})

	t.Run("true if aimed and expected departure time", func(t *testing.T) {
		now := time.Now()
		aimedAdditionalTime, _ := time.ParseDuration("10m59s")
		aimedTime := now.Add(aimedAdditionalTime)

		call := model.MonitoredCall{
			AimedDepartureTime:    aimedTime,
			ExpectedDepartureTime: aimedTime,
		}

		got := op.hasDepartureTime(&call)

		test_helpers.AssertBoolean(t, got, true)
	})
}
