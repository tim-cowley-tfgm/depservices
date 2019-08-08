package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/nationalrail"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/TfGMEnterprise/departures-service/test_helpers"
	"github.com/alicebob/miniredis"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"testing"
	"time"
)

const (
	locationAtcocode = "9100MNCRPIC"
)

var (
	locLondon, _ = time.LoadLocation("Europe/Paris")
	now          = time.Now().In(locLondon)
)

func buildSnsEvent(t *testing.T, stationBoard *nationalrail.StationBoard) events.SNSEvent {
	t.Helper()

	stationBoardJSON, err := json.Marshal(&stationBoard)
	if err != nil {
		t.Fatal(err)
	}

	event := events.SNSEvent{
		Records: []events.SNSEventRecord{
			{
				SNS: events.SNSEntity{
					Message: string(stationBoardJSON),
				},
			},
		},
	}

	return event
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

func TestRailngester_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("creates new data in the cache", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("MAN"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "2m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("14"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "via Bree",
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "8m").Format("15:04")),
							Etd:          createTimeType("Delayed"),
							Platform:     createPlatformType("13"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service2"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Minas Tirith"),
									Crs:          createCRSType("MNT"),
								},
								{
									LocationName: createLocationNameType("Isengard"),
									Crs:          createCRSType("ISN"),
									Via:          "via Hobbiton",
								},
							},
						},
					},
				},
			},
		}

		expectation1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service1",
			AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("On time"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("14"),
			Destination:        "Mordor via Bree",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		expectation2, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service2",
			AimedDepartureTime: test_helpers.AdjustTime(now, "8m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("Delayed"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("13"),
			Destination:        "Minas Tirith + Isengard via Hobbiton",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(expectation1), string(expectation2)}...)
	})

	t.Run("updates superseded data in the cache", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("MAN"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "2m").Format("15:04")),
							Etd:          createTimeType(test_helpers.AdjustTime(now, "5m").Format("15:04")),
							Platform:     createPlatformType("14"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "via Bree",
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "8m").Format("15:04")),
							Etd:          createTimeType("Cancelled"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service2"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Minas Tirith"),
									Crs:          createCRSType("MNT"),
								},
								{
									LocationName: createLocationNameType("Isengard"),
									Crs:          createCRSType("ISN"),
									Via:          "via Hobbiton",
								},
							},
						},
					},
				},
			},
		}

		seed1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service1",
			AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("On time"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("14"),
			Destination:        "Mordor via Bree",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		seed2, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service2",
			AimedDepartureTime: test_helpers.AdjustTime(now, "8m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("Delayed"),
			Stand:              aws.String("13"),
			LocationAtcocode:   locationAtcocode,
			Destination:        "Minas Tirith + Isengard via Hobbiton",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		expectation1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service1",
			AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String(test_helpers.AdjustTime(now, "5m").Format("15:04")),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("14"),
			Destination:        "Mordor via Bree",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		expectation2, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service2",
			AimedDepartureTime: test_helpers.AdjustTime(now, "8m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("Cancelled"),
			LocationAtcocode:   locationAtcocode,
			Destination:        "Minas Tirith + Isengard via Hobbiton",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{string(seed1), string(seed2)}...); err != nil {
			t.Fatal(err)
		}

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(expectation1), string(expectation2)}...)
	})

	t.Run("removes expired data from the cache", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("MAN"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "2m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("14"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "via Bree",
								},
							},
						},
					},
				},
			},
		}

		seed1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service0",
			AimedDepartureTime: test_helpers.AdjustTime(now, "-2m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("On time"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("13"),
			Destination:        "Hobbiton",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		expectation1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service1",
			AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("On time"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("14"),
			Destination:        "Mordor via Bree",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		if _, err := departuresDB.Push(locationAtcocode, []string{string(seed1)}...); err != nil {
			t.Fatal(err)
		}

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(expectation1)}...)
	})

	t.Run("removes expired data from source before calling any caches", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("MAN"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "-1m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("14"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "via Bree",
								},
							},
						},
					},
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "8m").Format("15:04")),
							Etd:          createTimeType("Delayed"),
							Platform:     createPlatformType("13"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service2"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Minas Tirith"),
									Crs:          createCRSType("MNT"),
								},
								{
									LocationName: createLocationNameType("Isengard"),
									Crs:          createCRSType("ISN"),
									Via:          "via Hobbiton",
								},
							},
						},
					},
				},
			},
		}

		expectation1, err := json.Marshal(model.Departure{
			RecordedAtTime:     now.Format(time.RFC3339),
			JourneyType:        model.Train,
			JourneyRef:         "Service2",
			AimedDepartureTime: test_helpers.AdjustTime(now, "8m").Truncate(time.Minute).Format(time.RFC3339),
			DepartureStatus:    aws.String("Delayed"),
			LocationAtcocode:   locationAtcocode,
			Stand:              aws.String("13"),
			Destination:        "Minas Tirith + Isengard via Hobbiton",
			OperatorCode:       "SR",
		})
		if err != nil {
			t.Fatal(err)
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}

		departuresDB.CheckList(t, locationAtcocode, []string{string(expectation1)}...)
	})

	t.Run("returns an error on departures connection failure", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("MAN"),
				PlatformAvailable: true,
			},
			TrainServices: &nationalrail.ArrayOfServiceItems{
				Service: []*nationalrail.ServiceItem{
					{
						BaseServiceItem: &nationalrail.BaseServiceItem{
							Std:          createTimeType(test_helpers.AdjustTime(now, "2m").Format("15:04")),
							Etd:          createTimeType("On time"),
							Platform:     createPlatformType("14"),
							Operator:     createTOCName("Sauron Rail"),
							OperatorCode: createTOCCode("SR"),
							ServiceID:    createServiceIDType("Service1"),
						},
						Destination: &nationalrail.ArrayOfServiceLocations{
							Location: []*nationalrail.ServiceLocation{
								{
									LocationName: createLocationNameType("Mordor"),
									Crs:          createCRSType("MDR"),
									Via:          "via Bree",
								},
							},
						},
					},
				},
			},
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", "")
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err == nil {
			t.Error("Should return an error!")
		}
	})

	t.Run("handles payloads which do not contain a TrainServices array", func(t *testing.T) {
		stationBoard := nationalrail.StationBoard{
			BaseStationBoard: &nationalrail.BaseStationBoard{
				GeneratedAt:       now,
				Crs:               createCRSType("WGW"),
				PlatformAvailable: true,
			},
		}

		departuresDB, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer departuresDB.Close()

		in := RailIngester{
			Logger: dlog.NewLogger([]dlog.LoggerOption{
				dlog.LoggerSetOutput(ioutil.Discard),
			}...),
			DeparturesPool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", departuresDB.Addr())
				}),
			}...),
			TimeLocation: locLondon,
		}

		event := buildSnsEvent(t, &stationBoard)

		if err := in.Handler(event); err != nil {
			t.Error(err)
			return
		}
	})
}

func TestRailIngester_convertDestination(t *testing.T) {
	in := RailIngester{}

	t.Run("single destination", func(t *testing.T) {
		input := &nationalrail.ArrayOfServiceLocations{
			Location: []*nationalrail.ServiceLocation{
				{
					LocationName: createLocationNameType("Hobbiton"),
				},
			},
		}

		got, err := in.convertDestination(input)
		if err != nil {
			t.Error(err)
		}

		want := "Hobbiton"

		if *got != want {
			t.Errorf("got %s, want %s", *got, want)
		}
	})

	t.Run("single destination with via point", func(t *testing.T) {
		input := &nationalrail.ArrayOfServiceLocations{
			Location: []*nationalrail.ServiceLocation{
				{
					LocationName: createLocationNameType("Hobbiton"),
					Via:          "via Bree",
				},
			},
		}

		got, err := in.convertDestination(input)
		if err != nil {
			t.Error(err)
		}

		want := "Hobbiton via Bree"

		if *got != want {
			t.Errorf("got %s, want %s", *got, want)
		}
	})

	t.Run("two destinations", func(t *testing.T) {
		input := &nationalrail.ArrayOfServiceLocations{
			Location: []*nationalrail.ServiceLocation{
				{
					LocationName: createLocationNameType("Hobbiton"),
				},
				{
					LocationName: createLocationNameType("Bree"),
				},
			},
		}

		got, err := in.convertDestination(input)
		if err != nil {
			t.Error(err)
		}

		want := "Hobbiton + Bree"

		if *got != want {
			t.Errorf("got %s, want %s", *got, want)
		}
	})

	t.Run("two destinations with via points", func(t *testing.T) {
		input := &nationalrail.ArrayOfServiceLocations{
			Location: []*nationalrail.ServiceLocation{
				{
					LocationName: createLocationNameType("Hobbiton"),
					Via:          "via Minas Tirith",
				},
				{
					LocationName: createLocationNameType("Bree"),
					Via:          "via Mordor",
				},
			},
		}

		got, err := in.convertDestination(input)
		if err != nil {
			t.Error(err)
		}

		want := "Hobbiton via Minas Tirith + Bree via Mordor"

		if *got != want {
			t.Errorf("got %s, want %s", *got, want)
		}
	})

	t.Run("three destinations", func(t *testing.T) {
		input := &nationalrail.ArrayOfServiceLocations{
			Location: []*nationalrail.ServiceLocation{
				{
					LocationName: createLocationNameType("Hobbiton"),
					Via:          "via Bree",
				},
				{
					LocationName: createLocationNameType("Mordor"),
				},
				{
					LocationName: createLocationNameType("Minas Tirith"),
					Via:          "via Osgiliath",
				},
			},
		}

		got, err := in.convertDestination(input)
		if err != nil {
			t.Error(err)
		}

		want := "Hobbiton via Bree + Mordor + Minas Tirith via Osgiliath"

		if *got != want {
			t.Errorf("got %s, want %s", *got, want)
		}
	})
}

func TestNREPoller_removeExpiredDepartures(t *testing.T) {
	in := RailIngester{
		TimeLocation: locLondon,
	}

	t.Run("on time departures", func(t *testing.T) {
		departures := model.Internal{
			Departures: []model.Departure{
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service1",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-1m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("On time"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service2",
					AimedDepartureTime: now.Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("On time"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service3",
					AimedDepartureTime: test_helpers.AdjustTime(now, "2m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("On time"),
				},
			},
		}

		err := in.removeExpiredDepartures(now, &departures)
		if err != nil {
			t.Error(err)
			return
		}

		if len(departures.Departures) != 2 {
			t.Errorf("expected 2 departures, got %d", len(departures.Departures))
			return
		}

		if departures.Departures[0].JourneyRef != "Service2" {
			t.Errorf("expected first service to have JourneyRef %s, got %s", "Service2", departures.Departures[0].JourneyRef)
			return
		}

		if departures.Departures[1].JourneyRef != "Service3" {
			t.Errorf("expected second service to have JourneyRef %s, got %s", "Service3", departures.Departures[1].JourneyRef)
			return
		}
	})

	t.Run("cancelled departures", func(t *testing.T) {
		departures := model.Internal{
			Departures: []model.Departure{
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service1",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-1m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Cancelled"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service2",
					AimedDepartureTime: now.Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Cancelled"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service3",
					AimedDepartureTime: test_helpers.AdjustTime(now, "1m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Cancelled"),
				},
			},
		}

		err := in.removeExpiredDepartures(now, &departures)
		if err != nil {
			t.Error(err)
			return
		}

		if len(departures.Departures) != 2 {
			t.Errorf("expected 2 departures, got %d", len(departures.Departures))
			return
		}

		if departures.Departures[0].JourneyRef != "Service2" {
			t.Errorf("expected first service to have JourneyRef %s, got %s", "Service2", departures.Departures[0].JourneyRef)
			return
		}

		if departures.Departures[1].JourneyRef != "Service3" {
			t.Errorf("expected second service to have JourneyRef %s, got %s", "Service3", departures.Departures[1].JourneyRef)
			return
		}
	})

	t.Run("delayed departures", func(t *testing.T) {
		departures := model.Internal{
			Departures: []model.Departure{
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service1",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-1m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Delayed"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service2",
					AimedDepartureTime: now.Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Delayed"),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service3",
					AimedDepartureTime: test_helpers.AdjustTime(now, "1m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String("Delayed"),
				},
			},
		}

		err := in.removeExpiredDepartures(now, &departures)
		if err != nil {
			t.Error(err)
			return
		}

		if len(departures.Departures) != 3 {
			t.Errorf("expected 3 departures, got %d", len(departures.Departures))
			return
		}

		if departures.Departures[0].JourneyRef != "Service1" {
			t.Errorf("expected first service to have JourneyRef %s, got %s", "Service1", departures.Departures[0].JourneyRef)
			return
		}

		if departures.Departures[1].JourneyRef != "Service2" {
			t.Errorf("expected second service to have JourneyRef %s, got %s", "Service2", departures.Departures[1].JourneyRef)
			return
		}

		if departures.Departures[2].JourneyRef != "Service3" {
			t.Errorf("expected third service to have JourneyRef %s, got %s", "Service3", departures.Departures[2].JourneyRef)
			return
		}
	})

	t.Run("late departures", func(t *testing.T) {
		departures := model.Internal{
			Departures: []model.Departure{
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service1",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-2m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String(test_helpers.AdjustTime(now, "-1m").Format("15:04")),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service2",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-2m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String(now.Format("15:04")),
				},
				{
					RecordedAtTime:     now.Format(time.RFC3339),
					JourneyType:        model.Train,
					JourneyRef:         "Service3",
					AimedDepartureTime: test_helpers.AdjustTime(now, "-2m").Truncate(time.Minute).Format(time.RFC3339),
					DepartureStatus:    aws.String(test_helpers.AdjustTime(now, "1m").Format("15:04")),
				},
			},
		}

		err := in.removeExpiredDepartures(now, &departures)
		if err != nil {
			t.Error(err)
			return
		}

		if len(departures.Departures) != 2 {
			t.Errorf("expected 2 departures, got %d", len(departures.Departures))
			return
		}

		if departures.Departures[0].JourneyRef != "Service2" {
			t.Errorf("expected first service to have JourneyRef %s, got %s", "Service2", departures.Departures[0].JourneyRef)
			return
		}

		if departures.Departures[1].JourneyRef != "Service3" {
			t.Errorf("expected second service to have JourneyRef %s, got %s", "Service3", departures.Departures[1].JourneyRef)
			return
		}
	})
}
