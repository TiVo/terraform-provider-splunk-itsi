package models

import (
	"fmt"
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
	log.Println(fmt.Sprintf("%q", x))
	return nil
}

func logResponse(req *http.Response) error {
	x, err := httputil.DumpResponse(req, true)
	if err != nil {
		return err
	}
	log.Println(fmt.Sprintf("%q", x))
	return nil
}
