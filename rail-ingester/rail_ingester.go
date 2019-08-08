package main

import (
	"encoding/json"
	"fmt"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/nationalrail"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type RailIngester struct {
	Logger         *dlog.Logger
	DeparturesPool *redis.Pool
	TimeLocation   *time.Location
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("rail-ingester: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	departuresRedisHost, exists := os.LookupEnv("DEPARTURES_REDIS_HOST")
	if !exists || departuresRedisHost == "" {
		logger.Fatal("DEPARTURES_REDIS_HOST not set in environment")
	}

	departuresPoolOptions := []repository.RedisPoolOption{
		repository.RedisPoolDial(func() (redis.Conn, error) {
			return redis.Dial("tcp", departuresRedisHost)
		}),
	}

	timeLocation, err := time.LoadLocation("Europe/London")
	if err != nil {
		logger.Fatal("cannot load time location for Europe/London")
	}

	in := RailIngester{
		Logger:         logger,
		DeparturesPool: repository.NewRedisPool(departuresPoolOptions...),
		TimeLocation:   timeLocation,
	}

	defer func() {
		in.Logger.Debug("close departures Redis pool")
		if err := in.DeparturesPool.Close(); err != nil {
			in.Logger.Print("failed to close departures Redis pool")
			return
		}
		in.Logger.Debug("closed departures Redis pool")
	}()

	lambda.Start(in.Handler)
}

func (in *RailIngester) Handler(event events.SNSEvent) error {
	in.Logger.Debug("Handler")

	done := make(chan struct{})
	defer close(done)

	errs := make(chan error)

	wg := sync.WaitGroup{}

	for _, records := range event.Records {
		wg.Add(1)

		go in.processRecords(done, &wg, records, errs)
	}

	go in.waitForProcessRecords(&wg, errs)

	// Handle any errors
	var gErr = false
	for err := range errs {
		in.Logger.Print(err)
		gErr = true
	}

	if gErr {
		return errors.New("an error occurred performing a concurrent operation: see previous log output")
	}

	in.Logger.Debug("Handler completed")

	return nil
}

func (in *RailIngester) waitForProcessRecords(wg *sync.WaitGroup, errs chan error) {
	wg.Wait()
	close(errs)
}

func (in *RailIngester) processRecords(exitImmediately <-chan struct{}, wg *sync.WaitGroup, records events.SNSEventRecord, errs chan error) {
	defer wg.Done()

	stationBoard := nationalrail.StationBoard{}

	err := json.Unmarshal([]byte(records.SNS.Message), &stationBoard)
	if err != nil {
		errs <- errors.Wrap(err, "could not unmarshal departures into a StationBoard")
		return
	}

	if stationBoard.Crs == nil {
		errs <- errors.New("StationBoard CRS code is nil")
		return
	}

	crs := string(*stationBoard.Crs)

	// Get the ATCO Code for the location
	atcocode, err := nationalrail.GetAtcoCode(crs)
	if err != nil {
		errs <- errors.Wrapf(err, "could not get ATCO code for %s", crs)
		return
	}

	// Transform station board into our internal departures model
	departures, err := in.transformToInternalModel(time.Now(), in.TimeLocation, &stationBoard, atcocode)
	if err != nil {
		errs <- errors.Wrapf(err, "could not transform response for %s", crs)
		return
	}

	// Remove any departures that have expired
	if err := in.removeExpiredDepartures(time.Now(), departures); err != nil {
		errs <- errors.Wrap(err, "could not remove expired departures from event data")
		return
	}

	// Cache departures for the station
	if err := in.updateCachedData(atcocode, departures); err != nil {
		errs <- errors.Wrap(err, "could not update cached data")
		return
	}

	select {
	case <-exitImmediately:
		errs <- errors.New("processing records cancelled")
		return
	default:
		return
	}
}

func (in *RailIngester) transformToInternalModel(now time.Time, localLocation *time.Location, stationBoard *nationalrail.StationBoard, locationAtcocode string) (*model.Internal, error) {
	in.Logger.Debug("transformToInternalModel")

	platformAvailable := stationBoard.PlatformAvailable

	departures := model.Internal{}

	if stationBoard.TrainServices != nil {
		for _, service := range stationBoard.TrainServices.Service {
			if service.ServiceID == nil {
				return nil, errors.New("ServiceID value is missing")
			}

			if service.Std == nil {
				return nil, fmt.Errorf("Std value is missing for %s", string(*service.ServiceID))
			}

			if service.Etd == nil {
				return nil, fmt.Errorf("Etd value is missing for %s", string(*service.ServiceID))
			}

			if service.Destination == nil {
				return nil, fmt.Errorf("Destination is missing for %s", string(*service.ServiceID))
			}

			if service.OperatorCode == nil {
				return nil, fmt.Errorf("OperatorCode is missing for %s", string(*service.ServiceID))
			}

			aimedDepartureTime, err := model.ConvertDepartureTime(&now, localLocation, string(*service.Std))
			if err != nil {
				return nil, errors.Wrapf(err, "cannot read departure time for %s", string(*service.ServiceID))
			}

			destination, err := in.convertDestination(service.Destination)
			if err != nil {
				return nil, errors.Wrapf(err, "cannot read destination for %s", string(*service.ServiceID))
			}

			departureStatus := string(*service.Etd)

			departure := model.Departure{
				RecordedAtTime:     stationBoard.GeneratedAt.Format(time.RFC3339),
				JourneyType:        model.Train,
				JourneyRef:         string(*service.ServiceID),
				AimedDepartureTime: aimedDepartureTime.Format(time.RFC3339),
				DepartureStatus:    &departureStatus,
				LocationAtcocode:   locationAtcocode,
				Destination:        *destination,
				OperatorCode:       string(*service.OperatorCode),
			}

			if platformAvailable && service.Platform != nil {
				platform := string(*service.Platform)
				departure.Stand = &platform
			}

			departures.Departures = append(departures.Departures, departure)
		}
	}

	return &departures, nil
}

func (in *RailIngester) convertDestination(locations *nationalrail.ArrayOfServiceLocations) (*string, error) {
	var destinations []string

	for _, location := range locations.Location {
		if location.LocationName == nil {
			return nil, errors.New("Empty location name in destination")
		}
		destination := string(*location.LocationName)

		if location.Via != "" {
			destination += " " + location.Via
		}

		destinations = append(destinations, destination)
	}

	combined := strings.Join(destinations, " + ")

	return &combined, nil
}

func (in *RailIngester) removeExpiredDepartures(now time.Time, departures *model.Internal) error {
	in.Logger.Debug("removeExpiredDepartures")

	i := 0
	for _, departure := range departures.Departures {
		if departure.IsExpired(now) {
			continue
		}

		departures.Departures[i] = departure
		i++
	}

	in.Logger.Debugf("removed %d expired departures", len(departures.Departures)-i)

	departures.Departures = departures.Departures[:i]

	return nil
}

func (in *RailIngester) updateCachedData(locationAtcocode string, departures *model.Internal) error {
	in.Logger.Debugf("updateCachedData for location `%s` (total %d departure(s))", locationAtcocode, len(departures.Departures))

	var err error = nil
	conn := in.DeparturesPool.Get()

	defer func() {
		in.Logger.Debug("close Redis connection")
		if cerr := conn.Close(); cerr != nil {
			err = cerr
			return
		}
		in.Logger.Debug("closed Redis connection successfully")
	}()

	if err := conn.Send("MULTI"); err != nil {
		return errors.Wrapf(err, "cannot initiate MULTI Redis transaction for location `%s`", locationAtcocode)
	}

	if err := conn.Send("DEL", locationAtcocode); err != nil {
		return errors.Wrapf(err, "cannot delete key `%s` in Redis database", locationAtcocode)
	}

	args := make([]interface{}, len(departures.Departures)+1)

	args[0] = locationAtcocode

	for i, departure := range departures.Departures {
		departureJSON, err := json.Marshal(departure)
		if err != nil {
			return errors.Wrapf(err, "cannot marshal JSON for departure `%s` at location `%s`", departure.JourneyRef, locationAtcocode)
		}
		args[i+1] = departureJSON
	}

	if len(args) > 1 {
		if err := conn.Send("RPUSH", args...); err != nil {
			return errors.Wrapf(err, "cannot store departures in Redis cache for location `%s`", locationAtcocode)
		}
	}

	if _, err := conn.Do("EXEC"); err != nil {
		return errors.Wrapf(err, "cannot execute Redis transaction for location `%s`", locationAtcocode)
	}

	return err
}
