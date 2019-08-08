package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/nationalrail"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/hooklift/gowsdl/soap"
	"github.com/pkg/errors"
	"log"
	"os"
)

type RailStation struct {
	CRSCode string `json:"crsCode"`
}

type NREPoller struct {
	Logger      *dlog.Logger
	Service     nationalrail.LDBServiceSoap
	SNSClient   snsiface.SNSAPI
	SNSTopicARN *string
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("rail-departures-board-poller: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	url, exists := os.LookupEnv("NRE_OPENLDBWS_URL")
	if !exists || url == "" {
		logger.Fatal("NRE_OPENLDBWS_URL not set in environment")
	}

	accessToken, exists := os.LookupEnv("NRE_OPENLDBWS_ACCESS_TOKEN")
	if !exists || accessToken == "" {
		logger.Fatal("NRE_OPENLDBWS_ACCESS_TOKEN not set in environment")
	}

	snsTopicURN, exists := os.LookupEnv("AWS_SNS_TOPIC_ARN")
	if !exists || snsTopicURN == "" {
		logger.Fatal("AWS_SNS_TOPIC_ARN not set in environment")
	}

	token := nationalrail.SOAPHeader{
		Header: nationalrail.AccessToken{
			TokenValue: accessToken,
		},
	}

	client := soap.NewClient(url)
	client.AddHeader(token)

	sess := session.Must(session.NewSession())

	snsClient := *sns.New(sess)

	nre := NREPoller{
		Logger:      logger,
		Service:     nationalrail.NewLDBServiceSoap(client),
		SNSClient:   &snsClient,
		SNSTopicARN: &snsTopicURN,
	}

	lambda.Start(nre.Handler)
}

func (nre NREPoller) Handler(railStation RailStation) error {
	nre.Logger.Debug("Handler")

	crs := nationalrail.CRSType(railStation.CRSCode)

	req := nationalrail.GetBoardRequestParams{
		Crs: &crs,
	}

	departureBoard, err := nre.Service.GetDepartureBoard(&req)
	if err != nil {
		return errors.Wrapf(err, "cannot get departure board for %s", railStation.CRSCode)
	}

	departuresJSON, err := json.Marshal(&departureBoard.GetStationBoardResult)
	if err != nil {
		return errors.Wrapf(err, "cannot marshal JSON from departure board for %s", railStation.CRSCode)
	}

	if _, err := nre.SNSClient.Publish(&sns.PublishInput{
		Message:  aws.String(string(departuresJSON)),
		TopicArn: nre.SNSTopicARN,
	}); err != nil {
		return errors.Wrapf(err, "cannot publish message to SNS topic `%s`", *nre.SNSTopicARN)
	}

	return nil
}
