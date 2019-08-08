package optis_client

import (
	"encoding/xml"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// OptisClient configuration options for connecting to and requesting information from OPTIS
type OptisClient struct {
	Client      *http.Client
	Logger      *dlog.Logger
	OptisURL    string
	OptisAPIKey string
}

type OptisClientInterface interface {
	Request(siriRequest string) (*model.Siri, int, error)
}

// Request makes the request to OPTIS and returns a SIRI struct representation of the data
func (o *OptisClient) Request(siriRequest string) (*model.Siri, int, error) {
	o.Logger.Debug("OPTIS Request")

	optisRequest, err := o.createOptisHTTPRequest(siriRequest)
	if err != nil {
		return nil, http.StatusBadRequest, errors.Wrap(err, "cannot create OPTIS HTTP request")
	}

	optisResponse, err := o.makeOptisHTTPRequest(*o.Client, *optisRequest)
	if err != nil {
		var statusCode int
		if optisResponse != nil {
			if optisResponse.StatusCode >= http.StatusInternalServerError {
				statusCode = http.StatusBadGateway
			} else {
				statusCode = optisResponse.StatusCode
			}
		} else {
			statusCode = http.StatusGatewayTimeout
		}
		return nil, statusCode, errors.Wrap(err, "cannot make OPTIS HTTP request")
	}

	rawSiriResponse, err := o.readOptisHTTPResponse(optisResponse)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Wrap(err, "cannot read OPTIS SIRI response")
	}

	siriResponse, err := o.createSiriResponseData(rawSiriResponse)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Wrap(err, "cannot unmarshal SIRI response")
	}

	if err := o.checkSiriResponseData(siriResponse); err != nil {
		return siriResponse, http.StatusBadRequest, errors.Wrap(err, "error returned from OPTIS")
	}

	return siriResponse, http.StatusOK, nil
}

func (o *OptisClient) createOptisHTTPRequest(siriRequest string) (*http.Request, error) {
	o.Logger.Debug("createOptisHTTPRequest")
	req, err := http.NewRequest("POST", o.OptisURL, strings.NewReader(siriRequest))
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("apiKey", o.OptisAPIKey)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "text/xml")
	return req, nil
}

func (o *OptisClient) makeOptisHTTPRequest(client http.Client, request http.Request) (*http.Response, error) {
	o.Logger.Debug("makeOptisHTTPRequest")
	resp, err := client.Do(&request)
	if err != nil {
		return nil, err
	}

	switch true {
	case resp.StatusCode >= http.StatusInternalServerError:
		return resp, errors.New("OPTIS is unavailable")
	case resp.StatusCode >= http.StatusBadRequest:
		return resp, errors.New("Bad request to OPTIS")
	default:
		return resp, nil
	}
}

func (o *OptisClient) readOptisHTTPResponse(response *http.Response) (body []byte, err error) {
	o.Logger.Debug("readOptisHTTPResponse")
	defer func() {
		o.Logger.Debug("close response")
		if ferr := response.Body.Close(); ferr != nil {
			err = ferr
			return
		}
		o.Logger.Debug("closed response successfully")
	}()

	body, err = ioutil.ReadAll(response.Body)
	return body, err
}

func (o *OptisClient) createSiriResponseData(body []byte) (*model.Siri, error) {
	o.Logger.Debug("createSiriResponseData")
	siriResponse := model.Siri{}
	err := xml.Unmarshal(body, &siriResponse)
	if err != nil {
		return nil, err
	}
	return &siriResponse, nil
}

func (o *OptisClient) checkSiriResponseData(siri *model.Siri) error {
	o.Logger.Debug("checkSiriResponseData")
	if siri.ServiceDelivery.ErrorCondition.Description != "" {
		return errors.New(siri.ServiceDelivery.ErrorCondition.Description)
	}
	if !siri.ServiceDelivery.Status {
		return errors.New("request failed with status == false")
	}

	return nil
}
