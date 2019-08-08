package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"github.com/rafaeljusto/redigomock"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func buildJSONDeparture(t *testing.T, recordedAtTime time.Time, journeyType model.JourneyType, journeyRef int, aimedDepartureTime time.Time, expectedDepartureTime *time.Time, locationAtcocode string, stand *string, destinationAtcocode string, destinationName string, serviceNumber string, operatorCode string) []byte {
	t.Helper()

	var expectedDepartureTimeStr *string
	if expectedDepartureTime != nil {
		expectedDepartureTimeRef := expectedDepartureTime.Format(time.RFC3339)
		expectedDepartureTimeStr = &expectedDepartureTimeRef
	} else {
		expectedDepartureTimeStr = nil
	}

	departure := model.Departure{
		RecordedAtTime:        recordedAtTime.Format(time.RFC3339),
		JourneyType:           journeyType,
		JourneyRef:            recordedAtTime.Format("2006-01-02") + `_` + strconv.Itoa(journeyRef),
		AimedDepartureTime:    aimedDepartureTime.Format(time.RFC3339),
		ExpectedDepartureTime: expectedDepartureTimeStr,
		LocationAtcocode:      locationAtcocode,
		Stand:                 stand,
		DestinationAtcocode:   destinationAtcocode,
		Destination:           destinationName,
		ServiceNumber:         serviceNumber,
		OperatorCode:          operatorCode,
	}

	departureJSON, err := json.Marshal(departure)
	if err != nil {
		t.Fatal(err)
	}

	return departureJSON
}

func TestPresenter_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	logger := dlog.NewLogger([]dlog.LoggerOption{
		dlog.LoggerSetOutput(ioutil.Discard),
	}...)

	t.Run("returns error if atcocode is not valid", func(t *testing.T) {
		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": "foo",
			},
		}

		p := &Presenter{
			Logger: logger,
		}

		_, err := p.Handler(req)

		if err == nil {
			t.Error("should return an error")
			return
		}

		if !strings.Contains(err.Error(), "foo") {
			t.Errorf("error should include the requested atcocode: %s", "foo")
		}
	})

	t.Run("returns error if top is not valid", func(t *testing.T) {
		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": "1800BNIN",
				"top":      "-1",
			},
		}

		p := &Presenter{
			Logger: logger,
		}

		_, err := p.Handler(req)

		if err == nil {
			t.Error("should return an error")
			return
		}

		if !strings.Contains(err.Error(), "-1") {
			t.Errorf("error should include the requested value: %d", -1)
		}
	})

	t.Run("gets the data for the requested bus atcocode from the cache", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		atcocode := "1800BNIN0C1"
		top := 4

		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": atcocode,
				"top":      strconv.Itoa(top),
			},
		}

		stand := "C"
		departure1ExpectedTime := test_helpers.AdjustTime(now, "50s")
		departure1 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1234,
			test_helpers.AdjustTime(now, "1m10s"),
			&departure1ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"123",
			"ANWE")
		departure2ExpectedTime := test_helpers.AdjustTime(now, "1m10s")
		departure2 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1235,
			test_helpers.AdjustTime(now, "4m10s"),
			&departure2ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"456",
			"ANWE")
		departure3ExpectedTime := test_helpers.AdjustTime(now, "2m10s")
		departure3 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1236,
			test_helpers.AdjustTime(now, "3m10s"),
			&departure3ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"789",
			"ANWE")
		departure4 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1237,
			test_helpers.AdjustTime(now, "5m10s"),
			nil,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"123",
			"ANWE")

		resp := []string{
			string(departure1),
			string(departure2),
			string(departure3),
			string(departure4),
		}

		conn := redigomock.NewConn()
		// Redis LRANGE returns upto and including the limit value;
		// i.e. LRANGE <key> 0 3 returns the first FOUR values
		conn.Command("LRANGE", atcocode, int64(0), int64(top)-1).ExpectStringSlice(resp...)

		p := &Presenter{
			Logger: logger,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return conn, nil
				}),
			}...),
		}

		got, err := p.Handler(req)
		if err != nil {
			t.Error(err)
			return
		}

		want := &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"content-type": "application/json",
			},
			Body: `{"journeyType":"` + string(model.Bus) + `","departures":[` +
				`{"departureTime":"Approaching","stand":"C","serviceNumber":"123","destination":"Hobbiton"},` +
				`{"departureTime":"1 min","stand":"C","serviceNumber":"456","destination":"Hobbiton"},` +
				`{"departureTime":"2 mins","stand":"C","serviceNumber":"789","destination":"Hobbiton"},` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "5m10s").Format("15:04") + `","stand":"C","serviceNumber":"123","destination":"Hobbiton"}` +
				`]}`,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected result: got %#v, wanted %#v\n", got, want)
		}
	})

	t.Run("sets a sensible default for the top value if none is provided", func(t *testing.T) {
		atcocode := "1800BNIN0C1"

		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": atcocode,
			},
		}

		conn := redigomock.NewConn()
		// Redis LRANGE returns upto and including the limit value;
		// i.e. LRANGE <key> 0 3 returns the first FOUR values
		conn.Command("LRANGE", atcocode, int64(0), int64(9)).ExpectStringSlice([]string{}...)

		p := &Presenter{
			Logger: logger,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return conn, nil
				}),
			}...),
		}

		_, err := p.Handler(req)
		if err != nil {
			t.Error(err)
			return
		}

		if err := conn.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
	})

	t.Run("removes expired departures and appends response with equivalent number of later departures", func(t *testing.T) {
		now := time.Now()
		atcocode := "1800BNIN0C1"

		top := 3
		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": atcocode,
				"top":      strconv.Itoa(top),
			},
		}

		stand := "C"

		departure1ExpectedTime := test_helpers.AdjustTime(now, "-5s")
		departure1 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1234,
			test_helpers.AdjustTime(now, "1m10s"),
			&departure1ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"123",
			"ANWE")
		departure2ExpectedTime := test_helpers.AdjustTime(now, "1m10s")
		departure2 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1235,
			test_helpers.AdjustTime(now, "4m10s"),
			&departure2ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"456",
			"ANWE")
		departure3ExpectedTime := test_helpers.AdjustTime(now, "2m10s")
		departure3 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1236,
			test_helpers.AdjustTime(now, "3m10s"),
			&departure3ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"789",
			"ANWE")
		departure4 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1237,
			test_helpers.AdjustTime(now, "5m10s"),
			nil,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"123",
			"ANWE")

		resp1 := []string{
			string(departure1),
			string(departure2),
			string(departure3),
		}

		resp2 := []string{
			string(departure4),
		}

		conn := redigomock.NewConn()
		conn.Command("LRANGE", atcocode, int64(0), int64(2)).ExpectStringSlice(resp1...)
		conn.Command("LRANGE", atcocode, int64(3), int64(3)).ExpectStringSlice(resp2...)

		p := &Presenter{
			Logger: logger,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return conn, nil
				}),
			}...),
		}

		got, err := p.Handler(req)
		if err != nil {
			t.Error(err)
			return
		}

		if err := conn.ExpectationsWereMet(); err != nil {
			t.Error(err)
			return
		}

		want := &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"content-type": "application/json",
			},
			Body: `{"journeyType":"` + string(model.Bus) + `","departures":[` +
				`{"departureTime":"1 min","stand":"C","serviceNumber":"456","destination":"Hobbiton"},` +
				`{"departureTime":"2 mins","stand":"C","serviceNumber":"789","destination":"Hobbiton"},` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "5m10s").Format("15:04") + `","stand":"C","serviceNumber":"123","destination":"Hobbiton"}` +
				`]}`,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected result: got %#v, wanted %#v\n", got, want)
		}
	})

	t.Run("stops making requests to Redis if there are no more departures to get", func(t *testing.T) {
		now := time.Now()
		atcocode := "1800BNIN0C1"
		top := 4

		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": atcocode,
				"top":      strconv.Itoa(top),
			},
		}

		stand := "C"

		departure1ExpectedTime := test_helpers.AdjustTime(now, "-5s")
		departure1 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1234,
			test_helpers.AdjustTime(now, "1m10s"),
			&departure1ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"123",
			"ANWE")
		departure2ExpectedTime := test_helpers.AdjustTime(now, "1m10s")
		departure2 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1235,
			test_helpers.AdjustTime(now, "4m10s"),
			&departure2ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"456",
			"ANWE")
		departure3ExpectedTime := test_helpers.AdjustTime(now, "2m10s")
		departure3 := buildJSONDeparture(
			t,
			test_helpers.AdjustTime(now, "-10s"),
			model.Bus,
			1236,
			test_helpers.AdjustTime(now, "3m10s"),
			&departure3ExpectedTime,
			atcocode,
			&stand,
			"1800WA12481",
			"Hobbiton",
			"789",
			"ANWE")

		resp := []string{
			string(departure1),
			string(departure2),
			string(departure3),
		}

		conn := redigomock.NewConn()
		conn.Command("LRANGE", atcocode, int64(0), int64(3)).ExpectStringSlice(resp...)
		conn.Command("LRANGE", atcocode, int64(4), int64(4)).ExpectStringSlice([]string{}...)

		p := &Presenter{
			Logger: logger,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return conn, nil
				}),
			}...),
		}

		got, err := p.Handler(req)
		if err != nil {
			t.Error(err)
			return
		}

		if err := conn.ExpectationsWereMet(); err != nil {
			t.Error(err)
			return
		}

		want := &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"content-type": "application/json",
			},
			Body: `{"journeyType":"` + string(model.Bus) + `","departures":[` +
				`{"departureTime":"1 min","stand":"C","serviceNumber":"456","destination":"Hobbiton"},` +
				`{"departureTime":"2 mins","stand":"C","serviceNumber":"789","destination":"Hobbiton"}` +
				`]}`,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected result: got %#v, wanted %#v\n", got, want)
		}
	})

	t.Run("gets the data for the requested train atcocode from the cache", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		atcocode := "9100MNCRPIC"
		top := 4

		req := events.APIGatewayProxyRequest{
			QueryStringParameters: map[string]string{
				"atcocode": atcocode,
				"top":      strconv.Itoa(top),
			},
		}

		departure1 := model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service1",
			AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Format(time.RFC3339),
			DepartureStatus:    aws.String("On time"),
			LocationAtcocode:   atcocode,
			Stand:              aws.String("1"),
			Destination:        "Hobbiton",
			OperatorCode:       "BR",
		}

		departure1JSON, err := json.Marshal(departure1)
		if err != nil {
			t.Fatal(err)
		}

		departure2 := model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service2",
			AimedDepartureTime: test_helpers.AdjustTime(now, "4m").Format(time.RFC3339),
			DepartureStatus:    aws.String("Delayed"),
			LocationAtcocode:   atcocode,
			Stand:              aws.String("2"),
			Destination:        "Mordor",
			OperatorCode:       "BR",
		}

		departure2JSON, err := json.Marshal(departure2)
		if err != nil {
			t.Fatal(err)
		}

		departure3 := model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service3",
			AimedDepartureTime: test_helpers.AdjustTime(now, "12m").Format(time.RFC3339),
			DepartureStatus:    aws.String("Cancelled"),
			LocationAtcocode:   atcocode,
			Destination:        "Minas Tirith",
			OperatorCode:       "BR",
		}

		departure3JSON, err := json.Marshal(departure3)
		if err != nil {
			t.Fatal(err)
		}

		departure4 := model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service4",
			AimedDepartureTime: test_helpers.AdjustTime(now, "15m").Format(time.RFC3339),
			DepartureStatus:    aws.String(test_helpers.AdjustTime(now, "20m").Format("15:04")),
			LocationAtcocode:   atcocode,
			Stand:              aws.String("4"),
			Destination:        "Hobbiton",
			OperatorCode:       "BR",
		}

		departure4JSON, err := json.Marshal(departure4)
		if err != nil {
			t.Fatal(err)
		}

		resp := []string{
			string(departure1JSON),
			string(departure2JSON),
			string(departure3JSON),
			string(departure4JSON),
		}

		conn := redigomock.NewConn()
		// Redis LRANGE returns upto and including the limit value;
		// i.e. LRANGE <key> 0 3 returns the first FOUR values
		conn.Command("LRANGE", atcocode, int64(0), int64(top)-1).ExpectStringSlice(resp...)

		p := &Presenter{
			Logger: logger,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return conn, nil
				}),
			}...),
		}

		got, err := p.Handler(req)
		if err != nil {
			t.Error(err)
			return
		}

		want := &events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"content-type": "application/json",
			},
			Body: `{"journeyType":"` + string(model.Train) + `","departures":[` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "2m").Format("15:04") + `","stand":"1","destination":"Hobbiton","departureStatus":"On time"},` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "4m").Format("15:04") + `","stand":"2","destination":"Mordor","departureStatus":"Delayed"},` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "12m").Format("15:04") + `","destination":"Minas Tirith","departureStatus":"Cancelled"},` +
				`{"departureTime":"` + test_helpers.AdjustTime(now, "15m").Format("15:04") + `","stand":"4","destination":"Hobbiton","departureStatus":"` + test_helpers.AdjustTime(now, "20m").Format("15:04") + `"}` +
				`]}`,
		}

		if !reflect.DeepEqual(got, want) {
			t.Errorf("unexpected result: got %#v, wanted %#v\n", got, want)
		}
	})

}
