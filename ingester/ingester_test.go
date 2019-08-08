package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/alicebob/miniredis"
	"github.com/aws/aws-lambda-go/events"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"strconv"
	"testing"
	"time"
)

const (
	locationAtcocode      = "1800BNIN0C1"
	extraLocationAtcocode = "1800BNIN0D1"
	stopAreaAtcocode      = "1800BNIN"
)

var (
	now                = time.Now()
	locationStand      = "C"
	extraLocationStand = "D"
)

func buildJSONDeparture(t *testing.T, recordedAtTime time.Time, journeyRef int, aimedDepartureTime time.Time, expectedDepartureTime *time.Time, locationAtcocode string, stand *string, destinationAtcocode string, destinationName string, serviceNumber string, operatorCode string) []byte {
	t.Helper()

	departure := model.Departure{
		RecordedAtTime:      recordedAtTime.Format(time.RFC3339),
		JourneyType:         model.Bus,
		JourneyRef:          serviceNumber + `_direction_` + recordedAtTime.Format("2006-01-02") + `_` + strconv.Itoa(journeyRef),
		AimedDepartureTime:  aimedDepartureTime.Format(time.RFC3339),
		LocationAtcocode:    locationAtcocode,
		DestinationAtcocode: destinationAtcocode,
		Destination:         destinationName,
		ServiceNumber:       serviceNumber,
		OperatorCode:        operatorCode,
	}

	if expectedDepartureTime != nil {
		expectedDepartureTimeRef := expectedDepartureTime.Format(time.RFC3339)
		expectedDepartureTimeStr := &expectedDepartureTimeRef
		departure.ExpectedDepartureTime = expectedDepartureTimeStr
	}

	if stand != nil {
		departure.Stand = stand
	}

	departureJSON, err := json.Marshal(departure)
	if err != nil {
		t.Fatal(err)
	}

	return departureJSON
}

func buildSnsEvent(t *testing.T, jsonDeps ...[]byte) events.SNSEvent {
	t.Helper()

	departures := model.Internal{}
	for i := 0; i < len(jsonDeps); i++ {
		dep := model.Departure{}
		if err := json.Unmarshal(jsonDeps[i], &dep); err != nil {
			t.Fatal(err)
		}
		departures.Departures = append(departures.Departures, dep)
	}

	departuresJSON, err := json.Marshal(&departures)
	if err != nil {
		t.Fatal(err)
	}

	event := events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				SNS: events.SNSEntity{
					Message: string(departuresJSON),
				},
			},
		},
	}

	return event
}

func TestIngester_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("updates superseded data in the cache", func(t *testing.T) {
		cachedExpectedDepartureTime1 := test_helpers.AdjustTime(now, "1m")
		cachedDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1234, test_helpers.AdjustTime(now, "-2m"), &cachedExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		newExpectedDepartureTime1 := test_helpers.AdjustTime(now, "2m")
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &newExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture1Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &newExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		cachedDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1235, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		newDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")
		newDeparture2Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{string(cachedDeparture1), string(cachedDeparture2)}...); err != nil {
			t.Fatal(err)
		}

		if _, err := departuresDB.Push(stopAreaAtcocode, []string{string(cachedDeparture1), string(cachedDeparture2)}...); err != nil {
			t.Fatal(err)
		}

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		if err := stopsInAreaDB.Set(locationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		if err := circularServicesDB.Set("VISB525", "Mordor circular"); err != nil {
			t.Fatal(err)
		}

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1, newDeparture2)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(newDeparture1Expectation), string(newDeparture2Expectation)}...)
		departuresDB.CheckList(t, stopAreaAtcocode, []string{string(newDeparture1Expectation), string(newDeparture2Expectation)}...)
	})

	t.Run("creates new data in the cache", func(t *testing.T) {
		newExpectedDepartureTime1 := test_helpers.AdjustTime(now, "1m")
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &newExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture1Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &newExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		newDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "2m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")
		newDeparture2Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "2m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		if err := stopsInAreaDB.Set(locationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		if err := circularServicesDB.Set("VISB525", "Mordor circular"); err != nil {
			t.Fatal(err)
		}

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1, newDeparture2)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(newDeparture1Expectation), string(newDeparture2Expectation)}...)
		departuresDB.CheckList(t, stopAreaAtcocode, []string{string(newDeparture1Expectation), string(newDeparture2Expectation)}...)
	})

	t.Run("removes expired data from the cache", func(t *testing.T) {
		cachedExpectedDepartureTime1 := test_helpers.AdjustTime(now, "-1m")
		cachedDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1234, test_helpers.AdjustTime(now, "-2m"), &cachedExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		cachedExpectedDepartureTime2 := test_helpers.AdjustTime(now, "1m")
		cachedDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1235, test_helpers.AdjustTime(now, "-2m"), &cachedExpectedDepartureTime2, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		cachedDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1236, test_helpers.AdjustTime(now, "-2m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")
		cachedDeparture4 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1237, test_helpers.AdjustTime(now, "2m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "VISB", "525")

		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture1Expectaton := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")
		newDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1239, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")
		newDeparture2Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1239, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{string(cachedDeparture1), string(cachedDeparture2), string(cachedDeparture3), string(cachedDeparture4)}...); err != nil {
			t.Fatal(err)
		}

		if _, err := departuresDB.Push(stopAreaAtcocode, []string{string(cachedDeparture1), string(cachedDeparture2), string(cachedDeparture3), string(cachedDeparture4)}...); err != nil {
			t.Fatal(err)
		}

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		if err := stopsInAreaDB.Set(locationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		if err := circularServicesDB.Set("VISB525", "Mordor circular"); err != nil {
			t.Fatal(err)
		}

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1, newDeparture2)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(cachedDeparture2), string(cachedDeparture4), string(newDeparture1Expectaton), string(newDeparture2Expectation)}...)
		departuresDB.CheckList(t, stopAreaAtcocode, []string{string(cachedDeparture2), string(cachedDeparture4), string(newDeparture1Expectaton), string(newDeparture2Expectation)}...)
	})

	t.Run("removes expired data from source before calling any caches", func(t *testing.T) {
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "-1m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1239, test_helpers.AdjustTime(now, "-1m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		if err := stopsInAreaDB.Set(locationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1, newDeparture2)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		if _, err := departuresDB.List(locationAtcocode); err == nil {
			t.Errorf("list for %s should be empty", locationAtcocode)
		}

		if _, err := departuresDB.List(stopAreaAtcocode); err == nil {
			t.Errorf("list for %s should be empty", stopAreaAtcocode)
		}
	})

	t.Run("handles data for multiple locations", func(t *testing.T) {
		// Should remove this
		location1CachedDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1233, test_helpers.AdjustTime(now, "-10s"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		// Should update this...
		location1CachedExpectedDepartureTime2 := test_helpers.AdjustTime(now, "-1m")
		location1CachedDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1234, test_helpers.AdjustTime(now, "-2m"), &location1CachedExpectedDepartureTime2, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")
		// ...with this
		location1NewExpectedDepartureTime2 := test_helpers.AdjustTime(now, "1m")
		location1NewDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &location1NewExpectedDepartureTime2, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		location1NewDeparture2Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1234, test_helpers.AdjustTime(now, "-2m"), &location1NewExpectedDepartureTime2, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		// Should update this...
		location1CachedDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1235, test_helpers.AdjustTime(now, "2m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")
		// ...with this
		location1NewDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")
		location1NewDeparture3Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1235, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		// Should create this
		location1NewDeparture4 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1236, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		location1NewDeparture4Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1236, test_helpers.AdjustTime(now, "4m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		// Should remove this...
		location2CachedDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1333, test_helpers.AdjustTime(now, "-1m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Bree circular", "561", "FMAN")
		// ...and this
		location2CachedDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1333, test_helpers.AdjustTime(now, "-1s"), nil, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Minas Tirith", "37", "SCMN")

		// Should update this...
		location2CachedExpectedDepartureTime3 := test_helpers.AdjustTime(now, "-1m")
		location2CachedDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1334, test_helpers.AdjustTime(now, "-1m"), &location2CachedExpectedDepartureTime3, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Minas Tirith", "37", "SCMN")
		// ...with this
		location2NewExpectedDepartureTime3 := test_helpers.AdjustTime(now, "2m")
		location2NewDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1334, test_helpers.AdjustTime(now, "-1m"), &location2NewExpectedDepartureTime3, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Parker Street", "37", "SCMN")
		location2NewDeparture3Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1334, test_helpers.AdjustTime(now, "-1m"), &location2NewExpectedDepartureTime3, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Minas Tirith", "37", "SCMN")

		// Should update this...
		location2CachedDeparture4 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1335, test_helpers.AdjustTime(now, "3m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Bree circular", "561", "FMAN")
		// ...with this
		location2NewDeparture4 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1335, test_helpers.AdjustTime(now, "4m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Hobbiton Interchange", "561", "FMAN")
		location2NewDeparture4Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1335, test_helpers.AdjustTime(now, "4m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Bree circular", "561", "FMAN")

		// Should create this...
		location2NewDeparture5 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1336, test_helpers.AdjustTime(now, "5m"), nil, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Parker Street", "37", "SCMN")
		location2NewDeparture5Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1336, test_helpers.AdjustTime(now, "5m"), nil, extraLocationAtcocode, &extraLocationStand, "1800SB45111", "Minas Tirith", "37", "SCMN")
		// ...and this
		location2NewDeparture6 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1337, test_helpers.AdjustTime(now, "6m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Hobbiton Interchange", "561", "FMAN")
		location2NewDeparture6Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0s"), 1337, test_helpers.AdjustTime(now, "6m"), nil, extraLocationAtcocode, &extraLocationStand, extraLocationAtcocode, "Bree circular", "561", "FMAN")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}
		if err := localityNamesDB.Set("1800SB45111", "Minas Tirith"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{
			string(location1CachedDeparture1),
			string(location1CachedDeparture2),
			string(location1CachedDeparture3),
		}...); err != nil {
			t.Fatal(err)
		}

		if _, err := departuresDB.Push(extraLocationAtcocode, []string{
			string(location2CachedDeparture1),
			string(location2CachedDeparture2),
			string(location2CachedDeparture3),
			string(location2CachedDeparture4),
		}...); err != nil {
			t.Fatal(err)
		}

		if _, err := departuresDB.Push(stopAreaAtcocode, []string{
			string(location1CachedDeparture1),
			string(location1CachedDeparture2),
			string(location1CachedDeparture3),
			string(location2CachedDeparture1),
			string(location2CachedDeparture2),
			string(location2CachedDeparture3),
			string(location2CachedDeparture4),
		}...); err != nil {
			t.Fatal(err)
		}

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		if err := stopsInAreaDB.Set(locationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		if err := stopsInAreaDB.Set(extraLocationAtcocode, stopAreaAtcocode); err != nil {
			t.Fatal(err)
		}

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		if err := circularServicesDB.Set("VISB525", "Mordor circular"); err != nil {
			t.Fatal(err)
		}

		if err := circularServicesDB.Set("FMAN561", "Bree circular"); err != nil {
			t.Fatal(err)
		}

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, location1NewDeparture2, location1NewDeparture3, location1NewDeparture4, location2NewDeparture3, location2NewDeparture4, location2NewDeparture5, location2NewDeparture6)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{
			string(location1NewDeparture2Expectation),
			string(location1NewDeparture3Expectation),
			string(location1NewDeparture4Expectation),
		}...)

		departuresDB.CheckList(t, extraLocationAtcocode, []string{
			string(location2NewDeparture3Expectation),
			string(location2NewDeparture4Expectation),
			string(location2NewDeparture5Expectation),
			string(location2NewDeparture6Expectation),
		}...)

		departuresDB.CheckList(t, stopAreaAtcocode, []string{
			string(location1NewDeparture2Expectation),
			string(location2NewDeparture3Expectation),
			string(location1NewDeparture3Expectation),
			string(location1NewDeparture4Expectation),
			string(location2NewDeparture4Expectation),
			string(location2NewDeparture5Expectation),
			string(location2NewDeparture6Expectation),
		}...)
	})

	t.Run("returns an error on departures connection failure", func(t *testing.T) {
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "1m"), nil, locationAtcocode, nil, "1800WA12481", "Turning Circle", "534", "ANWE")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1)

		if err := in.Handler(event); err == nil {
			t.Error("Should return an error!")
		}
	})

	t.Run("returns an error on stops in area connection failure", func(t *testing.T) {
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "1m"), nil, locationAtcocode, nil, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture1Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "1m"), nil, locationAtcocode, nil, "1800WA12481", "Hobbiton", "534", "ANWE")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		if err := localityNamesDB.Set("1800WA12481", "Hobbiton"); err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{string(newDeparture1)}...); err != nil {
			t.Fatal(err)
		}

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1)

		if err := in.Handler(event); err == nil {
			t.Error("Should return an error!")
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(newDeparture1Expectation)}...)
		if _, err := departuresDB.List(stopAreaAtcocode); err == nil {
			t.Errorf("Expected error for %s", stopAreaAtcocode)
		}
	})

	t.Run("handles stops that are not in a stop area", func(t *testing.T) {
		cachedExpectedDepartureTime1 := test_helpers.AdjustTime(now, "-1m")
		cachedDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1234, test_helpers.AdjustTime(now, "-2m"), &cachedExpectedDepartureTime1, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")

		cachedExpectedDepartureTime2 := test_helpers.AdjustTime(now, "1m")
		cachedDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1235, test_helpers.AdjustTime(now, "-2m"), &cachedExpectedDepartureTime2, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		cachedDeparture3 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1236, test_helpers.AdjustTime(now, "-2m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Hobbiton", "534", "ANWE")
		cachedDeparture4 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "-3m"), 1237, test_helpers.AdjustTime(now, "2m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "-1m"), nil, locationAtcocode, &locationStand, "1800WA12481", "Turning Circle", "534", "ANWE")
		newDeparture2 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1239, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Hobbiton Interchange", "525", "VISB")
		newDeparture2Expectation := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1239, test_helpers.AdjustTime(now, "3m"), nil, locationAtcocode, &locationStand, locationAtcocode, "Mordor circular", "525", "VISB")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{
			string(cachedDeparture1),
			string(cachedDeparture2),
			string(cachedDeparture3),
			string(cachedDeparture4),
		}...); err != nil {
			t.Fatal(err)
		}

		if _, err := departuresDB.Push(locationAtcocode, []string{string(newDeparture1)}...); err != nil {
			t.Fatal(err)
		}

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		if err := circularServicesDB.Set("VISB525", "Mordor circular"); err != nil {
			t.Fatal(err)
		}

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", localityNamesDB.Addr())
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1, newDeparture2)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{
			string(cachedDeparture2),
			string(cachedDeparture4),
			string(newDeparture2Expectation),
		}...)
	})

	t.Run("returns an error on locality names connection failure", func(t *testing.T) {
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "1m"), nil, locationAtcocode, nil, "1800WA12481", "Turning Circle", "534", "ANWE")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", circularServicesDB.Addr())
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1)

		if err := in.Handler(event); err == nil {
			t.Error("Should return an error!")
		}
	})

	t.Run("returns an error on circular services connection failure", func(t *testing.T) {
		newDeparture1 := buildJSONDeparture(t, test_helpers.AdjustTime(now, "0m"), 1238, test_helpers.AdjustTime(now, "1m"), nil, locationAtcocode, nil, "1800WA12481", "Turning Circle", "534", "ANWE")

		localityNamesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer localityNamesDB.Close()

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		stopsInAreaDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer stopsInAreaDB.Close()

		circularServicesDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer circularServicesDB.Close()

		in := Ingester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			LocalityNamesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			StopsInAreaPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", stopsInAreaDB.Addr())
				}),
			}...),
			CircularServicesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			circularServices: make(map[string]*string),
			localityNames:    make(map[string]*string),
			stopsInArea:      make(map[string]*string),
		}

		event := buildSnsEvent(t, newDeparture1)

		if err := in.Handler(event); err == nil {
			t.Error("Should return an error!")
		}
	})
}
