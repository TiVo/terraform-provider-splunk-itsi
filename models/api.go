package models

import (
	"github.com/tivo/terraform-provider-splunk-itsi/util"
)

var clients IHttpClients

var itsiLimiter *util.Limiter

func InitItsiApiLimiter(concurrency int) {
	itsiLimiter = util.NewLimiter(concurrency)
}
