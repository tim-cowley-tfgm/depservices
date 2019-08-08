package main

import (
	"errors"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/nationalrail"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/alicebob/miniredis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"io/ioutil"
	"reflect"
	"testing"
	"time"
)

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

func createLocationNameType(locationNameStr string) *nationalrail.LocationNameType {
	locationName := nationalrail.LocationNameType(locationNameStr)

	return &locationName
}

func createCRSType(crsStr string) *nationalrail.CRSType {
	crs := nationalrail.CRSType(crsStr)

	return &crs
}

func createTimeType(timeStr string) *nationalrail.TimeType {
	timeType := nationalrail.TimeType(timeStr)

	return &timeType
}

func createPlatformType(platformStr string) *nationalrail.PlatformType {
	platform := nationalrail.PlatformType(platformStr)

	return &platform
}

func createTOCCode(tocCodeStr string) *nationalrail.TOCCode {
	tocCode := nationalrail.TOCCode(tocCodeStr)

	return &tocCode
}

func createTOCName(tocNameStr string) *nationalrail.TOCName {
	tocName := nationalrail.TOCName(tocNameStr)

	return &tocName
}

func createServiceIDType(serviceIDStr string) *nationalrail.ServiceIDType {
	serviceIDType := nationalrail.ServiceIDType(serviceIDStr)

	return &serviceIDType
}

const (
	snsTopicArn = "arn:aws:sns:mars-north-8:123456789012:nre-departures"
)

var (
	now = time.Now()

	HappyRailStationResponse = &nationalrail.StationBoardResponseType{
		GetStationBoardResult: &nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				LocationName:      createLocationNameType("Hobbiton"),
				Crs:               createCRSType("HOB"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "-30m").Format("15:04")),
							Etd:          createTimeType("Delayed"),
							Platform:     createPlatformType("1"),
							Operator:     createTOCName("Broken and Late"),
							OperatorCode: createTOCCode("BL"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "Bree",
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "-5m").Format("15:04")),
							Etd:          createTimeType(test_helpers.AdjustTime(now, "5m").Format("15:04")),
							Platform:     createPlatformType("2"),
							Operator:     createTOCName("Tardy Trains"),
							OperatorCode: createTOCCode("TT"),
							ServiceID:    createServiceIDType("Service2"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Minas Tirith"),
									Crs:          createCRSType("MNT"),
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "2m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("3"),
							Operator:     createTOCName("Punctual Rail"),
							OperatorCode: createTOCCode("PR"),
							ServiceID:    createServiceIDType("Service3"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Bree"),
									Crs:          createCRSType("BRE"),
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "10m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("4"),
							Operator:     createTOCName("Sauron Splitter"),
							OperatorCode: createTOCCode("SS"),
							ServiceID:    createServiceIDType("Service4"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Angmar"),
									Crs:          createCRSType("ANG"),
								},
								{
									LocationName: createLocationNameType("Isengard"),
									Crs:          createCRSType("ISN"),
									Via:          "Mirkwood",
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "15m").Format("15:04")),
							Etd:          createTimeType("Cancelled"),
							Operator:     createTOCName("Broken and Late"),
							OperatorCode: createTOCCode("BL"),
							ServiceID:    createServiceIDType("Service5"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mount Doom"),
									Crs:          createCRSType("MTD"),
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "23m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("BUS"),
							Operator:     createTOCName("Broken and Late"),
							OperatorCode: createTOCCode("BL"),
							ServiceID:    createServiceIDType("Service6"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Osgiliath"),
									Crs:          createCRSType("OSG"),
								},
							},
						},
					},
				},
			},
		},
	}
)

type MockNREService struct {
	nationalrail.LDBServiceSoap
}

func (nre MockNREService) GetDepartureBoard(request *nationalrail.GetBoardRequestParams) (*nationalrail.StationBoardResponseType, error) {
	if *request.Crs == "HOB" {
		return HappyRailStationResponse, nil
	}

	return nil, errors.New("Something went wrong")
}

func TestNREPoller_Handler(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		mockedNREService := &MockNREService{}

		mockedSNSClient := &MockSNSClient{
			Output: sns.PublishOutput{
				MessageId: aws.String("ABC-123"),
			},
			T: t,
		}

		railStation := RailStation{
			CRSCode: "HOB",
		}

		railReferencesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer railReferencesDB.Close()

		if err := railReferencesDB.Set("HOB", "9100HBTN"); err != nil {
			t.Fatal(err)
		}

		nre := NREPoller{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			Service:     mockedNREService,
			SNSClient:   mockedSNSClient,
			SNSTopicARN: aws.String(snsTopicArn),
		}

		expectation := sns.PublishInput{
			Message: aws.String(`{` +
				`"generatedAt":"` + now.Format(time.RFC3339Nano) + `",` +
				`"locationName":"Hobbiton",` +
				`"crs":"HOB",` +
				`"platformAvailable":true,` +
				`"trainServices":{` +
				`"service":[` +
				`{"std":"` + test_helpers.AdjustTime(now, "-30m").Truncate(time.Minute).Format("15:04") + `","etd":"Delayed","platform":"1","operator":"Broken and Late","operatorCode":"BL","serviceID":"Service1","destination":{"location":[{"locationName":"Mordor","crs":"MDR","via":"Bree"}]}},` +
				`{"std":"` + test_helpers.AdjustTime(now, "-5m").Truncate(time.Minute).Format("15:04") + `","etd":"` + test_helpers.AdjustTime(now, "5m").Format("15:04") + `","platform":"2","operator":"Tardy Trains","operatorCode":"TT","serviceID":"Service2","destination":{"location":[{"locationName":"Minas Tirith","crs":"MNT"}]}},` +
				`{"std":"` + test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format("15:04") + `","etd":"On time","platform":"3","operator":"Punctual Rail","operatorCode":"PR","serviceID":"Service3","destination":{"location":[{"locationName":"Bree","crs":"BRE"}]}},` +
				`{"std":"` + test_helpers.AdjustTime(now, "10m").Truncate(time.Minute).Format("15:04") + `","etd":"On time","platform":"4","operator":"Sauron Splitter","operatorCode":"SS","serviceID":"Service4","destination":{"location":[{"locationName":"Angmar","crs":"ANG"},{"locationName":"Isengard","crs":"ISN","via":"Mirkwood"}]}},` +
				`{"std":"` + test_helpers.AdjustTime(now, "15m").Truncate(time.Minute).Format("15:04") + `","etd":"Cancelled","operator":"Broken and Late","operatorCode":"BL","serviceID":"Service5","destination":{"location":[{"locationName":"Mount Doom","crs":"MTD"}]}},` +
				`{"std":"` + test_helpers.AdjustTime(now, "23m").Truncate(time.Minute).Format("15:04") + `","etd":"On time","platform":"BUS","operator":"Broken and Late","operatorCode":"BL","serviceID":"Service6","destination":{"location":[{"locationName":"Osgiliath","crs":"OSG"}]}}` +
				`]}}`),
			TopicArn: aws.String(snsTopicArn),
		}

		mockedSNSClient.SetPublishExpectation(expectation)

		if err := nre.Handler(railStation); err != nil {
			t.Error(err)
			return
		}

		if mockedSNSClient.PublishCallCount != 1 {
			t.Error("SNS Publish should have been called once.")
		}
	})
}
