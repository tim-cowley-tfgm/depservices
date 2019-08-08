package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/model"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/TfGMEnterprise/departures-service/transxchange"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Counter struct {
	Count int
	mux   sync.Mutex
}

type CircularServices struct {
	Bucket        string
	Logger        *dlog.Logger
	Prefix        *string
	FilesSkipped  *Counter
	RecordsAdded  *Counter
	RedisPipeline *repository.RedisPipeline
	SearchTerms   []string
	TransXChange  *transxchange.TransXChange
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("circular-services: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	redisHost, exists := os.LookupEnv("CIRCULAR_SERVICES_REDIS_HOST")
	if !exists || redisHost == "" {
		logger.Fatal("CIRCULAR_SERVICES_REDIS_HOST not set in environment")
	}

	redisMaxActiveStr, exists := os.LookupEnv("CIRCULAR_SERVICES_REDIS_MAX_ACTIVE")
	if !exists || redisMaxActiveStr == "" {
		redisMaxActiveStr = "10"
	}

	flushAfterStr, exists := os.LookupEnv("FLUSH_AFTER")
	if !exists || flushAfterStr == "" {
		flushAfterStr = "10000"
	}

	flushAfter, err := strconv.Atoi(flushAfterStr)
	if err != nil {
		logger.Fatal("FLUSH_AFTER value is invalid")
	}

	if flushAfter < 1 {
		logger.Fatal("FLUSH_AFTER value must be greater than 0")
	}

	redisMaxActive, err := strconv.Atoi(redisMaxActiveStr)
	if err != nil {
		logger.Fatal("CIRCULAR_SERVICES_REDIS_MAX_ACTIVE value is invalid")
	}

	if redisMaxActive < 1 {
		logger.Fatal("CIRCULAR_SERVICES_REDIS_MAX_ACTIVE value must be greater than 0")
	}

	bucket, exists := os.LookupEnv("TXC_S3_BUCKET")
	if !exists || bucket == "" {
		logger.Fatal("TXC_S3_BUCKET not set in environment")
	}

	var prefix *string
	prefixString, exists := os.LookupEnv("TXC_S3_PREFIX")
	if exists || prefixString != "" {
		prefix = &prefixString
	}

	searchTermsString, exists := os.LookupEnv("SEARCH_TERMS")
	if !exists || searchTermsString == "" {
		logger.Fatal("SEARCH_TERMS not set in environment")
	}

	searchTerms := strings.Split(searchTermsString, ";")

	awsSess, err := session.NewSession()
	if err != nil {
		logger.Fatal("cannot initialize AWS session")
	}

	txc := transxchange.TransXChange{
		Client: s3.New(awsSess),
		Logger: logger,
	}

	cs := CircularServices{
		Bucket:       bucket,
		Logger:       logger,
		Prefix:       prefix,
		FilesSkipped: &Counter{},
		RecordsAdded: &Counter{},
		RedisPipeline: &repository.RedisPipeline{
			FlushAfter: flushAfter,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", redisHost)
				}),
				repository.RedisPoolMaxActive(redisMaxActive),
			}...),
		},
		SearchTerms:  searchTerms,
		TransXChange: &txc,
	}

	defer func() {
		cs.Logger.Debug("close Redis pool")
		if err := cs.RedisPipeline.Pool.Close(); err != nil {
			cs.Logger.Print("failed to close Redis pool")
			return
		}
		cs.Logger.Debug("closed Redis pool")
	}()

	lambda.Start(cs.Handler)
}

func (cs *CircularServices) Handler() error {
	cs.Logger.Debug("Handler")

	// Get latest TXC zip file from S3
	zipReader, err := cs.TransXChange.Download(cs.Bucket, cs.Prefix)
	if err != nil {
		cs.Logger.Fatal("cannot download TransXChange data")
	}

	// Handler closes the exitImmediately channel when it returns; it may do so before
	// receiving all the values from send and errs
	exitImmediately := make(chan struct{})
	defer close(exitImmediately)

	redisPipelineDone := make(chan struct{})
	send := make(chan repository.RedisCommand, cs.RedisPipeline.FlushAfter)
	errs := make(chan error)

	go cs.processTransXChangeFiles(exitImmediately, zipReader, send, errs)

	go cs.RedisPipeline.Pipeline(exitImmediately, redisPipelineDone, send, errs)

	go cs.waitForRedisPipeline(redisPipelineDone, errs)

	// Handle any errors
	var gErr = false
	for err := range errs {
		cs.Logger.Print(err)
		gErr = true
	}

	if gErr {
		return errors.New("an error occurred performing a concurrent operation: see previous log output")
	}

	cs.RecordsAdded.mux.Lock()
	defer cs.RecordsAdded.mux.Unlock()
	cs.Logger.Printf("stored %d record(s) in the Redis cache", cs.RecordsAdded.Count)
	cs.FilesSkipped.mux.Lock()
	defer cs.FilesSkipped.mux.Unlock()
	cs.Logger.Printf("skipped %d file(s)", cs.FilesSkipped.Count)

	return nil
}

// Waits for the Redis pipeline to complete and then closes the errs channel
func (cs *CircularServices) waitForRedisPipeline(done <-chan struct{}, errs chan error) {
	<-done
	close(errs)
}

func (cs *CircularServices) processTransXChangeFiles(exitImmediately <-chan struct{}, zipReader *zip.Reader, send chan repository.RedisCommand, errs chan error) {
	defer close(send)

	wg := sync.WaitGroup{}

	today := time.Now().Truncate(time.Hour * 24)

	cs.Logger.Printf("number of TransXChange files in zip: %d", len(zipReader.File))

	for _, zf := range zipReader.File {
		wg.Add(1)

		go cs.processTransXChangeFile(exitImmediately, &wg, zf, today, send, errs)
	}

	wg.Wait()
}

func (cs *CircularServices) processTransXChangeFile(exitImmediately <-chan struct{}, wg *sync.WaitGroup, zf *zip.File, today time.Time, send chan repository.RedisCommand, errs chan error) {
	cs.Logger.Debugf("processing TransXChange file %s", zf.Name)

	defer wg.Done()

	f, err := zf.Open()
	if err != nil {
		errs <- errors.Wrapf(err, "cannot open zipped file %s", zf.Name)
		return
	}

	defer func() {
		if err := f.Close(); err != nil {
			errs <- errors.Wrapf(err, "cannot close zipped file %s", zf.Name)
		}
	}()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		errs <- errors.Wrapf(err, "cannot read zipped file %s", zf.Name)
		return
	}

	txc := model.TransXChange{}
	if err := xml.Unmarshal(b, &txc); err != nil {
		if err == io.EOF && len(b) == 0 {
			cs.Logger.Debugf("skipping empty file %s", zf.Name)
			cs.FilesSkipped.mux.Lock()
			cs.FilesSkipped.Count++
			cs.FilesSkipped.mux.Unlock()
			return
		}
		errs <- errors.Wrapf(err, "cannot unmarshal XML from file %s", zf.Name)
		return
	}

	// Determine if TXC is for a circular service
	for _, service := range txc.Services.Service {
		// Check the operating period for the file to see if it relates
		// to the present time
		startDate, err := time.Parse("2006-01-02", service.OperatingPeriod.StartDate)
		if err != nil {
			errs <- errors.Wrapf(err, "cannot parse start date")
			return
		}

		if startDate.After(today) {
			cs.Logger.Debugf("skipping file %s: start date is in the future (%s)", zf.Name, service.OperatingPeriod.StartDate)
			continue
		}

		// If there is no end date, the TXC is assumed to be valid forever
		// into the future
		if service.OperatingPeriod.EndDate != "" {
			endDate, err := time.Parse("2006-01-02", service.OperatingPeriod.EndDate)
			if err != nil {
				errs <- errors.Wrap(err, "cannot parse end date")
				return
			}

			if endDate.Before(today) {
				cs.Logger.Debugf("skipping file %s: end date is in the past (%s)", zf.Name, service.OperatingPeriod.EndDate)
				continue
			}
		}

		// Check if the service description contains a search term
		serviceDescription := strings.TrimRight(service.Description, " ")

		for _, searchTerm := range cs.SearchTerms {
			re, err := regexp.Compile(`(?i)\b` + strings.ToLower(searchTerm) + `\b`)
			if err != nil {
				errs <- errors.Wrapf(err, "cannot perform regexp MatchString for term %s in file %s", searchTerm, zf.Name)
				return
			}

			match := re.MatchString(serviceDescription)

			// No match, continue
			if !match {
				cs.Logger.Debugf("search term `%s` not found in service description `%s` in file %s", searchTerm, serviceDescription, zf.Name)
				continue
			}

			cs.Logger.Debugf("search term `%s` found in service description `%s` in file %s", searchTerm, serviceDescription, zf.Name)

			// Match found
			// Store the description in the Redis cache for each
			// combination of line name and operator code
			// (in reality, this is likely to be a single combination)
			formattedServiceDescription := re.ReplaceAllString(serviceDescription, searchTerm)

			if len(service.Lines.Line) == 0 {
				cs.Logger.Printf("No line information found in file %s", zf.Name)
			}

			for _, line := range service.Lines.Line {
				actions := 0
				cs.Logger.Debugf("Adding command for %s (%s) in file %s", line.LineName, serviceDescription, zf.Name)
				for _, operator := range txc.Operators.LicensedOperator {
					cs.Logger.Debugf("%s (%s) operator type is %s in file %s", line.LineName, serviceDescription, "LicensedOperator", zf.Name)
					cs.addCommandToSendChan(exitImmediately, send, zf.Name, operator.OperatorCode, line.LineName, formattedServiceDescription, errs)
					actions++
				}

				for _, operator := range txc.Operators.Operator {
					cs.Logger.Debugf("%s (%s) operator type is %s in file %s", line.LineName, serviceDescription, "LicensedOperator", zf.Name)
					cs.addCommandToSendChan(exitImmediately, send, zf.Name, operator.OperatorCode, line.LineName, formattedServiceDescription, errs)
					actions++
				}

				if actions == 0 {
					cs.Logger.Printf("No operator information found for line %s in file %s", line.LineName, zf.Name)
				}
			}

			// Search term was found, so this file has been processed - return
			return
		}
	}

	// No action was taken in the services loop; record this
	cs.FilesSkipped.mux.Lock()
	cs.FilesSkipped.Count++
	cs.FilesSkipped.mux.Unlock()
}

func (cs *CircularServices) addCommandToSendChan(exitImmediately <-chan struct{}, send chan repository.RedisCommand, filename string, operatorCode string, lineName string, serviceDescription string, errs chan error) {
	key := operatorCode + lineName
	value := serviceDescription

	cs.Logger.Debugf("addCommandToSendChan - %s: %s", key, value)
	cs.RecordsAdded.mux.Lock()
	cs.RecordsAdded.Count++
	cs.RecordsAdded.mux.Unlock()

	select {
	case send <- repository.RedisCommand{
		Name:   "SET",
		Args:   []interface{}{key, value},
		Result: make(chan repository.RedisResult, 1),
	}:
	case <-exitImmediately:
		errs <- fmt.Errorf("cancelled processing file %s", filename)
	}
}
