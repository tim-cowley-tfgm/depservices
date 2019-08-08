package main

import (
	"bytes"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/TfGMEnterprise/departures-service/transxchange"
	"github.com/alicebob/miniredis"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"testing"
	"time"
)

const (
	S3_BUCKET_NAME = "foo"
	S3_PREFIX      = "/bar/"
	S3_OBJECT_KEY  = "txc.zip"
)

type mockedS3Client struct {
	s3iface.S3API
	Output s3.GetObjectOutput
}

func (m mockedS3Client) ListObjectsV2(input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	obj := s3.Object{
		Key:          aws.String(S3_PREFIX + S3_OBJECT_KEY),
		LastModified: aws.Time(time.Now()),
	}

	resp := s3.ListObjectsV2Output{
		Contents: []*s3.Object{
			&obj,
		},
		IsTruncated: aws.Bool(false),
	}

	return &resp, nil
}

func (m mockedS3Client) GetObject(*s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &m.Output, nil
}

func TestCircularServices_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("happy path", func(t *testing.T) {
		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		r, err := ioutil.ReadFile("../test_resources/txc.zip")
		if err != nil {
			t.Fatal(err)
		}

		m := mockedS3Client{
			Output: s3.GetObjectOutput{
				Body:          ioutil.NopCloser(bytes.NewReader(r)),
				ContentLength: aws.Int64(int64(len(r))),
			},
		}

		txc := transxchange.TransXChange{
			Client: m,
			Logger: logger,
		}

		prefix := S3_PREFIX

		cs := CircularServices{
			Bucket:       S3_BUCKET_NAME,
			Logger:       logger,
			Prefix:       &prefix,
			FilesSkipped: &Counter{},
			RecordsAdded: &Counter{},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 3,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", s.Addr())
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
			SearchTerms:  []string{"circular", "Metroshuttle"},
			TransXChange: &txc,
		}

		if err := cs.Handler(); err != nil {
			t.Error(err)
			return
		}

		if cs.RecordsAdded.Count != 4 {
			t.Errorf("expected RecordsAdded to be %d, got %d", 4, cs.RecordsAdded.Count)
			return
		}

		if cs.FilesSkipped.Count != 3 {
			t.Errorf("expected FilesSkipped to be %d, got %d", 3, cs.FilesSkipped.Count)
			return
		}

		s.CheckGet(t, "SCMN12", "Middleton - Moorclose circular via Mordor")
		s.CheckGet(t, "MCTR232", "Ashton - Hurst Cross - Broadoak circular")
		s.CheckGet(t, "VISB500", "Bolton Town Centre Metroshuttle")
		s.CheckGet(t, "VISB525", "Hall i'th Wood circular")
	})
}
