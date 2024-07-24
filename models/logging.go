package models

import (
	"log"
	"net/http"
	"net/http/httputil"
)

var Verbose bool = false

func logRequest(req *http.Request) error {
	x, err := httputil.DumpRequest(req, true)
	if err != nil {
		return err
	}
	log.Printf("%q\n", x)
	return nil
}

func logResponse(req *http.Response) error {
	x, err := httputil.DumpResponse(req, true)
	if err != nil {
		return err
	}
	log.Printf("%q\n", x)
	return nil
}
