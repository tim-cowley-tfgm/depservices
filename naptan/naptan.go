package naptan

import (
	"archive/zip"
	"bytes"
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type Naptan struct {
	Client *http.Client
	Logger *dlog.Logger
	URL    string
}

func (n *Naptan) Download() (*zip.Reader, error) {
	n.Logger.Debugf("Download NaPTAN data from %s", n.URL)

	var err error = nil
	resp, err := n.Client.Get(n.URL)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot download NaPTAN CSV from %s", n.URL)
	}

	defer func() {
		n.Logger.Debugf("closing connection to %s", n.URL)
		if rErr := resp.Body.Close(); rErr != nil {
			err = errors.Wrapf(rErr, "cannot close connection to %s", n.URL)
			return
		}
		n.Logger.Debugf("closed connection to %s", n.URL)
	}()

	if resp.StatusCode >= 400 {
		return nil, errors.Errorf("error response from %s - status code %d", n.URL, resp.StatusCode)
	}

	zipFile, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read zip file in response from %s", n.URL)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipFile), int64(len(zipFile)))
	if err != nil {
		return nil, errors.Wrap(err, "cannot read NaPTANcsv ZIP file")
	}

	return zipReader, err
}
