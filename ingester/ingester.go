package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

type Ingester struct {
	Logger               *dlog.Logger
	DeparturesPool       *redis.Pool
	LocalityNamesPool    *redis.Pool
	StopsInAreaPool      *redis.Pool
	CircularServicesPool *redis.Pool
	IngesterInterface
	circularServices map[string]*string
	localityNames    map[string]*string
	stopsInArea      map[string]*string
}

type IngesterInterface interface {
	Handler(departures *model.Internal) error
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("ingester: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	departuresRedisHost, exists := os.LookupEnv("DEPARTURES_REDIS_HOST")
	if !exists || departuresRedisHost == "" {
		logger.Fatal("DEPARTURES_REDIS_HOST not set in environment")
	}

	localityNamesRedisHost, exists := os.LookupEnv("LOCALITY_NAMES_REDIS_HOST")
	if !exists || localityNamesRedisHost == "" {
		logger.Fatal("LOCALITY_NAMES_REDIS_HOST not set in environment")
	}

	stopsInAreaRedisHost, exists := os.LookupEnv("STOPS_IN_AREA_REDIS_HOST")
	if !exists || stopsInAreaRedisHost == "" {
		logger.Fatal("STOPS_IN_AREA_REDIS_HOST not set in environment")
	}

	circularServicesRedisHost, exists := os.LookupEnv("CIRCULAR_SERVICES_REDIS_HOST")
	if !exists || circularServicesRedisHost == "" {
		logger.Fatal("CIRCULAR_SERVICES_REDIS_HOST not set in environment")
	}

	departuresPoolOptions := []repository.RedisPoolOption{
		repository.RedisPoolDial(func() (redis.Conn, error) {
			return redis.Dial("tcp", departuresRedisHost)
		}),
	}

	localityNamesPoolOptions := []repository.RedisPoolOption{
		repository.RedisPoolDial(func() (redis.Conn, error) {
			return redis.Dial("tcp", localityNamesRedisHost)
		}),
	}

	stopsInAreaPoolOptions := []repository.RedisPoolOption{
		repository.RedisPoolDial(func() (redis.Conn, error) {
			return redis.Dial("tcp", stopsInAreaRedisHost)
		}),
	}

	circularServicesPoolOptions := []repository.RedisPoolOption{
		repository.RedisPoolDial(func() (redis.Conn, error) {
			return redis.Dial("tcp", circularServicesRedisHost)
		}),
	}

	in := Ingester{
		Logger:               logger,
		DeparturesPool:       repository.NewRedisPool(departuresPoolOptions...),
		LocalityNamesPool:    repository.NewRedisPool(localityNamesPoolOptions...),
		StopsInAreaPool:      repository.NewRedisPool(stopsInAreaPoolOptions...),
		CircularServicesPool: repository.NewRedisPool(circularServicesPoolOptions...),
		circularServices:     make(map[string]*string),
		localityNames:        make(map[string]*string),
		stopsInArea:          make(map[string]*string),
	}

	defer func() {
		in.Logger.Debug("close departures Redis pool")
		if err := in.DeparturesPool.Close(); err != nil {
			in.Logger.Print("failed to close departures Redis pool")
			return
		}
		in.Logger.Debug("closed departures Redis pool")
	}()

	defer func() {
		in.Logger.Debug("close locality names Redis pool")
		if err := in.LocalityNamesPool.Close(); err != nil {
			in.Logger.Print("failed to close locality names Redis pool")
			return
		}
		in.Logger.Debug("closed locality names Redis pool")
	}()

	defer func() {
		in.Logger.Debug("close stops in area Redis pool")
		if err := in.StopsInAreaPool.Close(); err != nil {
			in.Logger.Print("failed to close stops in area Redis pool")
			return
		}
		in.Logger.Debug("closed stops in area Redis pool")
	}()

	defer func() {
		in.Logger.Debug("close circular services Redis pool")
		if err := in.CircularServicesPool.Close(); err != nil {
			in.Logger.Print("failed to close circular services Redis pool")
			return
		}
		in.Logger.Debug("closed circular services Redis pool")
	}()

	lambda.Start(in.Handler)
}

func (in Ingester) Handler(event events.SNSEvent) error {
	in.Logger.Debug("Handler")

	done := make(chan struct{})
	defer close(done)

	errs := make(chan error)

	wg := sync.WaitGroup{}

	for _, records := range event.Records {
		wg.Add(1)

		go func(done <-chan struct{}, errs chan error, records events.SNSEventRecord) {
			defer wg.Done()

			newDepartures := model.Internal{}

			err := json.Unmarshal([]byte(records.SNS.Message), &newDepartures)
			if err != nil {
				errs <- errors.Wrap(err, "could not unmarshal new departures")
				return
			}

			if err := in.removeExpiredDepartures(time.Now(), &newDepartures); err != nil {
				errs <- errors.Wrap(err, "could not remove expired departures from event data")
				return
			}

			if err := in.updateDestinationNames(&newDepartures); err != nil {
				errs <- errors.Wrap(err, "cannot update destination names")
				return
			}

			stopsDone := make(chan struct{})
			stopAreasDone := make(chan struct{})

			// Cache departures for stops
			go func(done <-chan struct{}, errs chan error, departures model.Internal) {
				defer close(stopsDone)

				groupedByStop := in.groupByStop(departures)

				swg := sync.WaitGroup{}

				for locationAtcocode, newDeparturesForLocation := range groupedByStop {
					swg.Add(1)

					go func(done <-chan struct{}, errs chan error, locationAtcocode string, newDeparturesForLocation []model.Departure) {
						defer swg.Done()

						if err := in.ingestLocation(locationAtcocode, &newDeparturesForLocation); err != nil {
							errs <- err
						}
					}(done, errs, locationAtcocode, newDeparturesForLocation)
				}

				swg.Wait()
			}(done, errs, newDepartures)

			// Cache departures for stop areas
			go func(done <-chan struct{}, errs chan error, departures model.Internal) {
				defer close(stopAreasDone)

				groupedByStopArea, err := in.groupByStopArea(departures)
				if err != nil {
					errs <- err
					return
				}

				swg := sync.WaitGroup{}

				for locationAtcocode, newDeparturesForLocation := range groupedByStopArea {
					swg.Add(1)

					go func(done <-chan struct{}, errs chan error, locationAtcocode string, newDeparturesForLocation []model.Departure) {
						defer swg.Done()

						if err := in.ingestLocation(locationAtcocode, &newDeparturesForLocation); err != nil {
							errs <- err
						}
					}(done, errs, locationAtcocode, newDeparturesForLocation)
				}

				swg.Wait()
			}(done, errs, newDepartures)

			<-stopsDone
			<-stopAreasDone

			select {
			case <-done:
				errs <- errors.New("processing records cancelled")
				return
			default:
				return
			}
		}(done, errs, records)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	// Handle any errors
	var gErr = false
	for err := range errs {
		if err != nil {
			in.Logger.Print(err)
			gErr = true
		}
	}

	if gErr {
		return errors.New("an error occurred performing a concurrent operation: see previous log output")
	}

	in.Logger.Debug("Handler completed")

	return nil
}

func (in Ingester) updateDestinationNames(departures *model.Internal) error {
	in.Logger.Debug("updateDestinationNames")

	for i, departure := range departures.Departures {
		circularServiceDestination, err := in.getCircularServiceDestination(departure.OperatorCode, departure.ServiceNumber)
		if err != nil {
			return errors.Wrapf(err, "cannot get circular service destination for service %s %s", departure.OperatorCode, departure.ServiceNumber)
		}

		if circularServiceDestination != nil {
			in.Logger.Debugf("service %s %s is a circular service", departure.OperatorCode, departure.ServiceNumber)
			departures.Departures[i].Destination = *circularServiceDestination
			continue
		}

		in.Logger.Debugf("service %s %s is a point-to-point service", departure.OperatorCode, departure.ServiceNumber)

		localityName, err := in.getLocalityName(departure.DestinationAtcocode)
		if err != nil {
			return errors.Wrapf(err, "cannot get locality name for ATCO code %s", departure.DestinationAtcocode)
		}

		if localityName != nil {
			departures.Departures[i].Destination = *localityName
			continue
		}

		in.Logger.Printf("destination not updated for service %s %s going to %s; output is %s", departure.OperatorCode, departure.ServiceNumber, departure.DestinationAtcocode, departure.Destination)
	}

	return nil
}

func (in Ingester) getCircularServiceDestination(operatorCode string, serviceNumber string) (*string, error) {
	in.Logger.Debugf("getCircularServiceDescription for %s %s", operatorCode, serviceNumber)

	key := operatorCode + serviceNumber

	if val, exists := in.circularServices[key]; exists {
		in.Logger.Debugf("got circular service destination `%v` for `%s` from local cache", val, key)
		return val, nil
	}

	var err error

	conn := in.CircularServicesPool.Get()

	defer func() {
		in.Logger.Debug("close circular services connection")
		if cErr := conn.Close(); cErr != nil {
			err = cErr
			return
		}
		in.Logger.Debug("closed circular services connection")
	}()

	circularServiceDestination, err := redis.String(conn.Do("GET", key))
	if err == redis.ErrNil {
		in.Logger.Debugf("no circular service destination for `%s` in Redis cache", key)

		in.circularServices[key] = nil

		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Store retrieved locality name in local cache
	in.circularServices[key] = &circularServiceDestination

	in.Logger.Debugf("got circular service destination `%s` for `%s` from Redis cache", circularServiceDestination, key)
	return &circularServiceDestination, err
}

func (in Ingester) getLocalityName(atcocode string) (*string, error) {
	in.Logger.Debugf("getLocalityName for %s", atcocode)

	if val, exists := in.localityNames[atcocode]; exists {
		in.Logger.Debugf("got locality name `%v` for `%s` from local cache", val, atcocode)
		return val, nil
	}

	var err error

	conn := in.LocalityNamesPool.Get()

	defer func() {
		in.Logger.Debug("close locality names connection")
		if cErr := conn.Close(); cErr != nil {
			err = cErr
			return
		}
		in.Logger.Debug("closed locality names connection")
	}()

	localityName, err := redis.String(conn.Do("GET", atcocode))
	if err == redis.ErrNil {
		in.Logger.Debugf("no locality name for ATCO code `%s` in Redis cache", atcocode)

		in.localityNames[atcocode] = nil

		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Store retrieved locality name in local cache
	in.localityNames[atcocode] = &localityName

	in.Logger.Debugf("got locality name `%s` for `%s` from Redis cache", localityName, atcocode)
	return &localityName, err
}

func (in Ingester) groupByStop(departures model.Internal) map[string][]model.Departure {
	in.Logger.Debug("groupByStop")

	groupedByStop := make(map[string][]model.Departure)

	for _, departure := range departures.Departures {
		groupedByStop[departure.LocationAtcocode] = append(groupedByStop[departure.LocationAtcocode], departure)
	}

	return groupedByStop
}

func (in Ingester) groupByStopArea(departures model.Internal) (map[string][]model.Departure, error) {
	in.Logger.Debug("groupByStopArea")
	var err error
	groupedByStopArea := make(map[string][]model.Departure)

	for _, departure := range departures.Departures {
		groupedByStopArea, err = in.appendDepartureToStopArea(groupedByStopArea, departure)
		if err != nil {
			return nil, errors.Wrap(err, "cannot group by stop area")
		}
	}

	return groupedByStopArea, err
}

func (in Ingester) getStopArea(atcocode string) (*string, error) {
	in.Logger.Debugf("getStopArea for %s", atcocode)

	if val, exists := in.stopsInArea[atcocode]; exists {
		in.Logger.Debugf("got stop area `%s` for `%s` from local cache", *val, atcocode)
		return val, nil
	}

	var err error

	conn := in.StopsInAreaPool.Get()

	defer func() {
		in.Logger.Debug("close stops in area connection")
		if cErr := conn.Close(); cErr != nil {
			err = cErr
			return
		}
		in.Logger.Debug("closed stops in area connection")
	}()

	stopArea, err := redis.String(conn.Do("GET", atcocode))
	if err == redis.ErrNil {
		in.Logger.Debugf("no stop area for ATCO code `%s` in Redis cache", atcocode)

		in.stopsInArea[atcocode] = nil

		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Store retrieved locality name in local cache
	in.stopsInArea[atcocode] = &stopArea

	in.Logger.Debugf("got stop area `%s` for `%s` from Redis cache", stopArea, atcocode)
	return &stopArea, err
}

func (in Ingester) appendDepartureToStopArea(groupedByStopArea map[string][]model.Departure, departure model.Departure) (map[string][]model.Departure, error) {
	in.Logger.Debug("appendDepartureToStopArea")

	stopArea, err := in.getStopArea(departure.LocationAtcocode)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get stop area for %s", departure.LocationAtcocode)
	}

	if stopArea == nil {
		in.Logger.Debugf("location `%s` is not in a stop area", departure.LocationAtcocode)

		return groupedByStopArea, nil
	}

	in.Logger.Debugf("location `%s` is in stop area `%s`", departure.LocationAtcocode, *stopArea)

	groupedByStopArea[*stopArea] = append(groupedByStopArea[*stopArea], departure)

	return groupedByStopArea, nil
}

func (in Ingester) ingestLocation(locationAtcocode string, newDepartures *[]model.Departure) error {
	in.Logger.Debugf("ingestLocation: `%s`", locationAtcocode)

	departures, err := in.getDeparturesFromCache(locationAtcocode)
	if err != nil {
		return err
	}

	in.combineCachedAndNewDepartures(departures, newDepartures)

	if err := in.removeExpiredDepartures(time.Now(), departures); err != nil {
		return errors.Wrap(err, "could not remove expired departures from combined data")
	}

	sort.Sort(model.ByDepartureTime(departures.Departures))

	if err := in.updateCachedData(locationAtcocode, departures); err != nil {
		return err
	}

	return nil
}

func (in Ingester) getDeparturesFromCache(locationAtcocode string) (*model.Internal, error) {
	in.Logger.Debugf("getDeparturesFromCache for location `%s`", locationAtcocode)

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

	cachedRecordsLength, err := conn.Do("LLEN", locationAtcocode)
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrapf(err, "cannot get cached record length for location `%s` from Redis", locationAtcocode)
	}

	in.Logger.Debugf("%d cached records in Redis for key `%s`", cachedRecordsLength, locationAtcocode)

	cachedDepartures := model.Internal{}

	if cachedRecordsLength.(int64) == 0 {
		return &cachedDepartures, nil
	}

	cachedRecords, err := redis.Strings(conn.Do("LRANGE", locationAtcocode, int64(0), cachedRecordsLength.(int64)-1))
	if err != nil && err != redis.ErrNil {
		return nil, errors.Wrapf(err, "cannot get cached records for location `%s` from Redis", locationAtcocode)
	}

	for _, departure := range cachedRecords {
		unmarshalledDeparture := model.Departure{}
		if err := json.Unmarshal([]byte(departure), &unmarshalledDeparture); err != nil {
			return nil, errors.Wrapf(err, "cannot unmarshal cached record for location `%s` from Redis", locationAtcocode)
		}
		cachedDepartures.Departures = append(cachedDepartures.Departures, unmarshalledDeparture)
	}

	return &cachedDepartures, err
}

func (in Ingester) combineCachedAndNewDepartures(departures *model.Internal, newDepartures *[]model.Departure) {
	in.Logger.Debug("combineCachedAndNewDepartures")

	newDeps := make(map[string]model.Departure)

	// Map new data by journey ref
	in.Logger.Debug("create new and updated departures map")
	for _, departure := range *newDepartures {
		in.Logger.Debugf("assign new and updated departure %s to map", departure.JourneyRef)
		newDeps[departure.JourneyRef] = departure
	}

	// Replace existing departures
	in.Logger.Debug("replace existing departures from cache with updated data")
	for i, departure := range departures.Departures {
		if newDep, ok := newDeps[departure.JourneyRef]; ok {
			in.Logger.Debugf("existing departure for %s; replacing", departure.JourneyRef)
			departures.Departures[i] = newDep
		}
		in.Logger.Debugf("delete updated departure %s from map", departure.JourneyRef)
		delete(newDeps, departure.JourneyRef)
	}

	// Append new departures
	in.Logger.Debugf("append new departures to cache - %d record(s)", len(newDeps))
	for _, departure := range newDeps {
		departures.Departures = append(departures.Departures, departure)
	}
}

func (in Ingester) removeExpiredDepartures(now time.Time, departures *model.Internal) error {
	in.Logger.Debug("removeExpiredDepartures")

	i := 0
	for _, departure := range departures.Departures {
		if departure.ExpectedDepartureTime != nil {
			expectedDepartureTime, err := time.Parse(time.RFC3339, *departure.ExpectedDepartureTime)
			if err != nil {
				return errors.Wrapf(err, "cannot parse expected departure time `%#v`", departure.ExpectedDepartureTime)
			}

			if !expectedDepartureTime.Before(now) {
				departures.Departures[i] = departure
				i++
			}
		} else {
			aimedDepartureTime, err := time.Parse(time.RFC3339, departure.AimedDepartureTime)
			if err != nil {
				return errors.Wrapf(err, "cannot parse aimed departure time `%#v`", departure.AimedDepartureTime)
			}

			if !aimedDepartureTime.Before(now) {
				departures.Departures[i] = departure
				i++
			}
		}
	}

	in.Logger.Debugf("removed %d expired departures", len(departures.Departures)-i)

	departures.Departures = departures.Departures[:i]

	return nil
}

func (in Ingester) updateCachedData(locationAtcocode string, departures *model.Internal) error {
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
