package transxchange

import (
	"bytes"
	"fmt"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/fortytw2/leaktest"
	"github.com/pkg/errors"
	"io/ioutil"
	"testing"
	"time"
)

const (
	S3_BUCKET_NAME       = "foo"
	S3_BUCKET_PATH       = "/bar/TransXChange/env/"
	S3_LAST_MODIFIED_KEY = "TFGM TXC 190617.zip"
)

type mockedS3Client struct {
	s3iface.S3API
	GetObjectOutput s3.GetObjectOutput
	ListOutputV2    s3.ListObjectsV2Output
}

func (m mockedS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	if *input.Key == S3_BUCKET_PATH+S3_LAST_MODIFIED_KEY {
		r, err := ioutil.ReadFile("../test_resources/txc.zip")
		if err != nil {
			return nil, err
		}

		output := s3.GetObjectOutput{
			Body:          ioutil.NopCloser(bytes.NewReader(r)),
			ContentLength: aws.Int64(int64(len(r))),
		}
		return &output, nil
	}

	return nil, fmt.Errorf("unexpected key %s in GetObject call", *input.Key)
}

func (m mockedS3Client) ListObjectsV2(input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
	if *input.Bucket != S3_BUCKET_NAME {
		return nil, errors.New("invalid bucket name")
	}
	if *input.Prefix != S3_BUCKET_PATH {
		return nil, errors.New("invalid bucket prefix")
	}

	objectKey1 := S3_BUCKET_PATH + "TFGM TXC 190606.zip"
	objectLastModified1 := time.Now().Add(time.Hour * -1)

	objectKey2 := S3_BUCKET_PATH + S3_LAST_MODIFIED_KEY
	objectLastModified2 := time.Now().Add(time.Minute * -2)

	objectKey3 := S3_BUCKET_PATH + "TFGM TXC 190508.foo"
	objectLastModified3 := time.Now().Add(time.Minute * -1)

	object1 := s3.Object{
		Key:          &objectKey1,
		LastModified: &objectLastModified1,
	}

	object2 := s3.Object{
		Key:          &objectKey2,
		LastModified: &objectLastModified2,
	}

	object3 := s3.Object{
		Key:          &objectKey3,
		LastModified: &objectLastModified3,
	}

	resp := s3.ListObjectsV2Output{}

	if input.ContinuationToken == nil {
		resp.Contents = []*s3.Object{
			&object1,
		}
		resp.IsTruncated = aws.Bool(true)
		resp.NextContinuationToken = aws.String("a")

		return &resp, nil
	}

	if *input.ContinuationToken == "a" {
		resp.Contents = []*s3.Object{
			&object2,
		}
		resp.IsTruncated = aws.Bool(true)
		resp.NextContinuationToken = aws.String("b")

		return &resp, nil
	}

	resp.Contents = []*s3.Object{
		&object3,
	}
	resp.IsTruncated = aws.Bool(false)

	return &resp, nil
}

func TestTransXChange_Download(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("returns the latest TransXChange zip file", func(t *testing.T) {
		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		txc := TransXChange{
			Client: mockedS3Client{},
			Logger: logger,
		}

		prefix := S3_BUCKET_PATH

		zipReader, err := txc.Download(S3_BUCKET_NAME, &prefix)
		if err != nil {
			t.Fatal(err)
		}

		var filesInZip []string

		for _, zf := range zipReader.File {
			filesInZip = append(filesInZip, zf.Name)
		}

		got := len(filesInZip)
		want := 7

		if got != want {
			t.Errorf("got %d files in zip; want %d", got, want)
		}
	})
}
