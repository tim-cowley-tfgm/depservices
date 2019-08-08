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

type LocalityNames struct {
	Filename      string
	Logger        *dlog.Logger
	NaptanClient  *naptan.Naptan
	RedisPipeline *repository.RedisPipeline
}

func main() {
	loggerOptions := []dlog.LoggerOption{
		dlog.LoggerSetOutput(os.Stderr),
		dlog.LoggerSetPrefix("locality-names: "),
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

	stopsFilename, exists := os.LookupEnv("NAPTAN_CSV_STOPS_FILENAME")
	if !exists || stopsFilename != "" {
		stopsFilename = "Stops.csv"
	}

	redisHost, exists := os.LookupEnv("LOCALITY_NAME_REDIS_HOST")
	if !exists || redisHost == "" {
		logger.Fatal("LOCALITY_NAME_REDIS_HOST not set in environment")
	}

	redisMaxActiveStr, exists := os.LookupEnv("LOCALITY_NAME_REDIS_MAX_ACTIVE")
	if !exists || redisMaxActiveStr == "" {
		redisMaxActiveStr = "10"
	}

	redisMaxActive, err := strconv.Atoi(redisMaxActiveStr)
	if err != nil {
		logger.Fatal("LOCALITY_NAME_REDIS_MAX_ACTIVE value is invalid")
	}

	if redisMaxActive < 1 {
		logger.Fatal("LOCALITY_NAME_REDIS_MAX_ACTIVE value must be greater than 0")
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

	ln := LocalityNames{
		Filename: stopsFilename,
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
		ln.Logger.Debug("close Redis pool")
		if err := ln.RedisPipeline.Pool.Close(); err != nil {
			ln.Logger.Print("failed to close Redis pool")
			return
		}
		ln.Logger.Debug("closed Redis pool")
	}()

	lambda.Start(ln.Handler)
}

// Handler downloads the NaPTANcsv zip file, extracts the relevant CSV file
// from within the ZIP and stores the relevant data in a Redis cache
func (ln *LocalityNames) Handler() error {
	// Download ZIP
	zipReader, err := ln.NaptanClient.Download()
	if err != nil {
		return errors.Wrap(err, "failed to download ZIP file")
	}

	// Handler closes the exitImmediately channel when it returns; it may do so before
	// receiving all the values from send and errs
	exitImmediately := make(chan struct{})
	defer close(exitImmediately)

	redisPipelineDone := make(chan struct{})
	send := make(chan repository.RedisCommand, ln.RedisPipeline.FlushAfter)
	errs := make(chan error)

	go ln.processZipFile(exitImmediately, zipReader, send, errs)

	go ln.RedisPipeline.Pipeline(exitImmediately, redisPipelineDone, send, errs)

	go ln.waitForRedisPipeline(redisPipelineDone, errs)

	// Handle any errors
	var gErr = false
	for err := range errs {
		ln.Logger.Print(err)
		gErr = true
	}

	if gErr {
		return errors.New("an error occurred performing a concurrent operation: see previous log output")
	}

	return nil
}

// Waits for the Redis pipeline to complete and then closes the errs channel
func (ln *LocalityNames) waitForRedisPipeline(done <-chan struct{}, errs chan error) {
	<-done
	close(errs)
}

// channelFileInZip reads the compressed StopsInArea.csv file and adds commands
// to set the data in a Redis cache
func (ln *LocalityNames) channelFileInZip(exitImmediately <-chan struct{}, zf *zip.File, send chan repository.RedisCommand, errs chan error) {
	// Close the send channel when the function exits
	defer func() {
		close(send)
	}()

	ln.Logger.Debugf("open file in ZIP: %s", zf.Name)
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

	ln.Logger.Debugf("read file in ZIP: %s", zf.Name)

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
			Args:   []interface{}{row[0], row[18]},
			Result: make(chan repository.RedisResult, 1),
		}:
		case <-exitImmediately:
			errs <- errors.New("channelFileInZip cancelled")
			return
		}
	}

	ln.Logger.Debug("Read file done")
}

// processZipFile loops through the NaPTANcsv.zip file and looks for the
// file that matches the configured target filename.
// If the file is found, it is passed to an extractor function which returns
// a "send" channel containing the commands to put the contents of the file in
// a Redis cache.
// If the file is not found, an error is created on the error channel and
// the send channel is immediately closed.
func (ln *LocalityNames) processZipFile(exitImmediately <-chan struct{}, zipReader *zip.Reader, send chan repository.RedisCommand, errs chan error) {
	ln.Logger.Debug("processZipFile")

	for _, zf := range zipReader.File {
		if strings.ToLower(zf.Name) != strings.ToLower(ln.Filename) {
			ln.Logger.Debugf("skip file in ZIP: %s", zf.Name)
			continue
		}

		go ln.channelFileInZip(exitImmediately, zf, send, errs)

		return
	}

	// File not found - nothing will get published to the send channel
	errs <- fmt.Errorf("file %s not found in ZIP", ln.Filename)
	close(send)
}
