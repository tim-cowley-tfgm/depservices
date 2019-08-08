package main

import (
	"encoding/json"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gomodule/redigo/redis"
)

type Presenter struct {
	Logger *dlog.Logger
	Pool   *redis.Pool
	PresenterInterface
}

type PresenterInterface interface {
	Handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error)
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("presenter: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	departuresRedisHost, exists := os.LookupEnv("DEPARTURES_REDIS_HOST")
	if !exists || departuresRedisHost == "" {
		logger.Fatal("DEPARTURES_REDIS_HOST not set in environment")
	}

	p := &Presenter{
		Logger: logger,
		Pool: repository.NewRedisPool([]repository.RedisPoolOption{
			repository.RedisPoolDial(func() (redis.Conn, error) {
				return redis.Dial("tcp", departuresRedisHost)
			}),
		}...),
	}

	defer func() {
		p.Logger.Debug("close Redis pool")
		if err := p.Pool.Close(); err != nil {
			p.Logger.Print("failed to close Redis pool")
			return
		}
		p.Logger.Debug("closed Redis pool")
	}()

	lambda.Start(p.Handler)
}

func (p Presenter) Handler(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	p.Logger.Debug("Handler")

	atcocode, exists := request.QueryStringParameters["atcocode"]
	if !exists {
		return nil, errors.New("atcocode is required")
	}

	topStr, exists := request.QueryStringParameters["top"]
	if !exists {
		// Set a sensible default for top if the value is not set in the request
		topStr = "10"
	}

	top, err := strconv.ParseInt(topStr, 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "top value `%s` is not valid", topStr)
	}

	// Validate Atcocode
	if !p.validateAtcocode(atcocode) {
		return nil, errors.Errorf("atcocode value `%s` is not valid", atcocode)
	}

	// Validate Top
	if !p.validateTop(top) {
		return nil, errors.Errorf("top value `%d` is not valid", top)
	}

	deps := model.Internal{}

	// Get data from Redis cache and remove expired departures, up to limit
	now := time.Now()
	start := int64(0)
	end := top

	for {
		if err := p.assignNextDepartures(&deps, atcocode, start, end); err != nil {
			return nil, err
		}

		removed := p.removeExpiredDepartures(now, &deps)

		if removed == 0 || len(deps.Departures) == int(top) {
			break
		}

		start += top
		end += removed
	}

	// Transform data for output purposes
	output := model.Output{
		JourneyType: model.GetJourneyType(atcocode),
	}

	for _, dep := range deps.Departures {
		depTime, err := p.transformDepartureTime(now, dep)
		if err != nil {
			return nil, err
		}

		depDisplay := model.DepartureDisplay{
			DepartureTime:   depTime,
			Stand:           dep.Stand,
			ServiceNumber:   dep.ServiceNumber,
			Destination:     dep.Destination,
			DepartureStatus: dep.DepartureStatus,
		}

		output.Departures = append(output.Departures, depDisplay)
	}

	// Marshal data in JSON format and return
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"content-type": "application/json",
		},
		Body: string(outputJSON),
	}, err
}

func (p Presenter) assignNextDepartures(departures *model.Internal, atcocode string, start int64, end int64) error {
	p.Logger.Debugf("assignNextDepartures for %s (start: %d; end: %d)", atcocode, start, end)

	var err error = nil
	conn := p.Pool.Get()
	defer func() {
		if cErr := conn.Close(); cErr != nil {
			err = cErr
		}
	}()

	cDeps, cErr := redis.Strings(conn.Do("LRANGE", atcocode, start, end-1))
	if cErr != nil && cErr == redis.ErrNil {
		return nil
	}

	if cErr != nil {
		return cErr
	}

	for i := 0; i < len(cDeps); i++ {
		dep := model.Departure{}
		if uErr := json.Unmarshal([]byte(cDeps[i]), &dep); uErr != nil {
			return uErr
		}
		departures.Departures = append(departures.Departures, dep)
	}
	return err
}
