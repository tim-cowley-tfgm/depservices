package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/naptan"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type StopsInArea struct {
	Filename      string
	Logger        *dlog.Logger
	NaptanClient  *naptan.Naptan
	RedisPipeline *repository.RedisPipeline
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("stops-in-area: "),
		dlog.LoggerSetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile),
	}

	logger := dlog.NewLogger(loggerOptions...)

	logger.Debug("main")

	naptanCSVDataSource, exists := os.LookupEnv("NAPTAN_CSV_DATA_SOURCE")
	if !exists || naptanCSVDataSource == "" {
		logger.Fatal("NAPTAN_CSV_DATA_SOURCE not set in environment")
	}

	naptanCSVTimeoutStr, exists := os.LookupEnv("NAPTAN_CSV_TIMEOUT")
	if !exists || naptanCSVTimeoutStr == "" {
		naptanCSVTimeoutStr = "60"
	}

	naptanCSVTimeout, err := strconv.Atoi(naptanCSVTimeoutStr)
	if err != nil {
		logger.Fatal("NAPTAN_CSV_TIMEOUT value is invalid")
	}

	if naptanCSVTimeout < 1 {
		logger.Fatal("NAPTAN_CSV_TIMEOUT value must be greater than 0")
	}

	stopsInAreaFileName, exists := os.LookupEnv("NAPTAN_CSV_STOPS_IN_AREA_FILENAME")
	if !exists || stopsInAreaFileName != "" {
		stopsInAreaFileName = "StopsInArea.csv"
	}

	redisHost, exists := os.LookupEnv("STOPS_IN_AREA_REDIS_HOST")
	if !exists || redisHost == "" {
		logger.Fatal("STOPS_IN_AREA_REDIS_HOST not set in environment")
	}

	redisMaxActiveStr, exists := os.LookupEnv("STOPS_IN_AREA_REDIS_MAX_ACTIVE")
	if !exists || redisMaxActiveStr == "" {
		redisMaxActiveStr = "10"
	}

	redisMaxActive, err := strconv.Atoi(redisMaxActiveStr)
	if err != nil {
		logger.Fatal("STOPS_IN_AREA_REDIS_MAX_ACTIVE value is invalid")
	}

	if redisMaxActive < 1 {
		logger.Fatal("STOPS_IN_AREA_REDIS_MAX_ACTIVE value must be greater than 0")
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

	sia := StopsInArea{
		Filename: stopsInAreaFileName,
		Logger:   logger,
		NaptanClient: &naptan.Naptan{
			Client: &http.Client{
				Timeout: time.Second * time.Duration(naptanCSVTimeout),
			},
			Logger: logger,
			URL:    naptanCSVDataSource,
		},
		RedisPipeline: &repository.RedisPipeline{
			FlushAfter: flushAfter,
			Pool: repository.NewRedisPool([]repository.RedisPoolOption{
				repository.RedisPoolDial(func() (redis.Conn, error) {
					return redis.Dial("tcp", redisHost)
				}),
				repository.RedisPoolMaxActive(redisMaxActive),
			}...),
		},
	}

	defer func() {
		sia.Logger.Debug("close Redis pool")
		if err := sia.RedisPipeline.Pool.Close(); err != nil {
			sia.Logger.Print("failed to close Redis pool")
			return
		}
		sia.Logger.Debug("closed Redis pool")
	}()

	lambda.Start(sia.Handler)
}

// Handler downloads the NaPTANcsv zip file, extracts the relevant CSV file
// from within the ZIP and stores the relevant data in a Redis cache
func (sia *StopsInArea) Handler() error {
	// Download ZIP
	zipReader, err := sia.NaptanClient.Download()
	if err != nil {
		return errors.Wrap(err, "failed to download ZIP file")
	}

	// Handler closes the exitImmediately channel when it returns; it may do so before
	// receiving all the values from send and errs
	exitImmediately := make(chan struct{})
	defer close(exitImmediately)

	redisPipelineDone := make(chan struct{})
	send := make(chan repository.RedisCommand, sia.RedisPipeline.FlushAfter)
	errs := make(chan error)

	go sia.processZipFile(exitImmediately, zipReader, send, errs)

	go sia.RedisPipeline.Pipeline(exitImmediately, redisPipelineDone, send, errs)

	go sia.waitForRedisPipeline(redisPipelineDone, errs)

	// Handle any errors
	var gErr = false
	for err := range errs {
		sia.Logger.Print(err)
		gErr = true
	}

	if gErr {
		return errors.New("an error occurred performing a concurrent operation: see previous log output")
	}

	return nil
}

// Waits for the Redis pipeline to complete and then closes the errs channel
func (sia *StopsInArea) waitForRedisPipeline(done <-chan struct{}, errs chan error) {
	<-done
	close(errs)
}

// channelFileInZip reads the compressed StopsInArea.csv file and adds commands
// to set the data in a Redis cache
func (sia *StopsInArea) channelFileInZip(exitImmediately <-chan struct{}, zf *zip.File, send chan repository.RedisCommand, errs chan error) {
	// Close the send channel when the function exits
	defer close(send)

	sia.Logger.Debugf("open file in ZIP: %s", zf.Name)
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

	sia.Logger.Debugf("read file in ZIP: %s", zf.Name)

	r := csv.NewReader(f)

	// Skip the header row
	if _, err := r.Read(); err != nil {
		errs <- errors.Wrapf(err, "cannot skip header row in file %s", zf.Name)
		return
	}

	// Loop through the rows in the CSV and add a command to set the in a
	// Redis database to the send channel
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errs <- errors.Wrapf(err, "cannot read row in file %s", zf.Name)
			return
		}
		select {
		case send <- repository.RedisCommand{
			Name:   "SET",
			Args:   []interface{}{row[1], row[0]},
			Result: make(chan repository.RedisResult, 1),
		}:
		case <-exitImmediately:
			errs <- errors.New("channelFileInZip cancelled")
			return
		}
	}
}

// processZipFile loops through the NaPTANcsv.zip file and looks for the
// file that matches the configured target filename.
// If the file is found, it is passed to an extractor function which returns
// a "send" channel containing the commands to put the contents of the file in
// a Redis cache.
// If the file is not found, an error is created on the error channel and
// the send channel is immediately closed.
func (sia *StopsInArea) processZipFile(exitImmediately <-chan struct{}, zipReader *zip.Reader, send chan repository.RedisCommand, errs chan error) {
	sia.Logger.Debug("processZipFile")

	for _, zf := range zipReader.File {
		if strings.ToLower(zf.Name) != strings.ToLower(sia.Filename) {
			sia.Logger.Debugf("skip file in ZIP: %s", zf.Name)
			continue
		}

		go sia.channelFileInZip(exitImmediately, zf, send, errs)

		return
	}

	// File not found - nothing will get published to the send channel
	errs <- fmt.Errorf("file %s not found in ZIP", sia.Filename)
	close(send)
}
