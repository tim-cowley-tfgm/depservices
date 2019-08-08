package main

import (
	"github.com/TfGMEnterprise/departures-service/dlog"
	"github.com/TfGMEnterprise/departures-service/naptan"
	"github.com/TfGMEnterprise/departures-service/repository"
	"github.com/alicebob/miniredis"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createNaptanSourceStub(t *testing.T) *httptest.Server {
	t.Helper()

	naptanSourceHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		zip, err := ioutil.ReadFile("../test_resources/NaPTANcsv.zip")
		if err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename='NaPTANcsv.zip'")
		if _, err := w.Write(zip); err != nil {
			t.Fatal(err)
		}
	}

	return httptest.NewServer(http.HandlerFunc(naptanSourceHandler))
}

func TestRailReferences_Handler(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("happy path", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		ln := LocalityNames{
			Filename: "RailReferences.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 1000,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", s.Addr())
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := ln.Handler(); err != nil {
			t.Error(err)
			return
		}

		s.CheckGet(t, "DGT", "9100MNCRDGT")
		s.CheckGet(t, "MCV", "9100MNCRVIC")
		s.CheckGet(t, "MCO", "9100MNCROXR")
		s.CheckGet(t, "ALD", "9100ALDEDGE")
		s.CheckGet(t, "HDG", "9100HLDG")
		s.CheckGet(t, "GTY", "9100GATLEY")
		s.CheckGet(t, "MAN", "9100MNCRPIC")
		s.CheckGet(t, "ADK", "9100ARDWICK")
	})

	t.Run("file not found in zip", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		ln := LocalityNames{
			Filename: "NonExistant.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 1000,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", s.Addr())
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := ln.Handler(); err == nil {
			t.Error("an error should have occurred")
			return
		}
	})

	t.Run("handles Redis connection failure", func(t *testing.T) {
		naptanStub := createNaptanSourceStub(t)
		defer naptanStub.Close()

		logger := dlog.NewLogger([]dlog.LoggerOption{
			dlog.LoggerSetOutput(ioutil.Discard),
		}...)

		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		ln := LocalityNames{
			Filename: "RailReferences.csv",
			Logger:   logger,
			NaptanClient: &naptan.Naptan{
				Client: naptanStub.Client(),
				Logger: logger,
				URL:    naptanStub.URL,
			},
			RedisPipeline: &repository.RedisPipeline{
				FlushAfter: 1000,
				Pool: repository.NewRedisPool([]repository.RedisPoolOption{
					repository.RedisPoolDial(func() (redis.Conn, error) {
						return redis.Dial("tcp", "")
					}),
					repository.RedisPoolMaxActive(10),
				}...),
			},
		}

		if err := ln.Handler(); err == nil {
			t.Error("an error should have occurred")
			return
		}
	})
}
