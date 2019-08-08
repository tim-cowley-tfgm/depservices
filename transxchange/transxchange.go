package transxchange

import (
	"archive/zip"
	"bytes"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pkg/errors"
	"io/ioutil"
	"regexp"
	"sort"
)

type TransXChange struct {
	Client s3iface.S3API
	Logger *dlog.Logger
}

func (txc TransXChange) Download(bucket string, prefix *string) (*zip.Reader, error) {
	txc.Logger.Debugf("Download TransXChange from %s", bucket)

	objects, err := txc.listZipObjects(bucket, prefix)
	if err != nil {
		return nil, err
	}

	obj, err := txc.getObject(bucket, txc.getLastModifiedObject(objects))
	if err != nil {
		return nil, err
	}

	zipFile, err := ioutil.ReadAll(obj.Body)
	if err != nil {
		return nil, err
	}

	return zip.NewReader(bytes.NewReader(zipFile), *obj.ContentLength)
}

func (txc TransXChange) listZipObjects(bucket string, prefix *string) ([]*s3.Object, error) {
	txc.Logger.Debugf("listZipObjects for bucket %s", bucket)
	var objects []*s3.Object

	req := s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: prefix,
	}

	re, err := regexp.Compile(`\.zip$`)
	if err != nil {
		return nil, errors.Wrap(err, "error compiling file extension regex")
	}

	// List objects in bucket
	for {
		resp, err := txc.Client.ListObjectsV2(&req)
		if err != nil {
			return nil, err
		}

		for _, object := range resp.Contents {
			if re.MatchString(*object.Key) {
				objects = append(objects, object)
			}
		}

		if !*resp.IsTruncated {
			txc.Logger.Debugf("All objects listed from bucket %s", bucket)
			break
		}

		txc.Logger.Debugf("More objects available in bucket %s", bucket)

		req.SetContinuationToken(*resp.NextContinuationToken)
	}

	return objects, nil
}

func (txc TransXChange) getObject(bucket string, object *s3.Object) (*s3.GetObjectOutput, error) {
	txc.Logger.Debugf("getObject %v from bucket %s", &object.Key, bucket)

	req := s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    object.Key,
	}

	return txc.Client.GetObject(&req)
}

func (txc TransXChange) getLastModifiedObject(objects []*s3.Object) *s3.Object {
	sort.Sort(ByLastModified(objects))

	return objects[len(objects)-1]
}

type ByLastModified []*s3.Object

func (a ByLastModified) Len() int {
	return len(a)
}

func (a ByLastModified) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByLastModified) Less(i, j int) bool {
	iLastModified := a[i].LastModified
	jLastModified := a[j].LastModified

	return iLastModified.Before(*jLastModified)
}
