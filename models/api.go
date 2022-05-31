package models

import (
	"github.com/tivo/terraform-provider-splunk-itsi/provider/util"
)

var clients IHttpClients

var itsiLimiter *util.Limiter

func InitItsiApiLimiter(concurrency int) {
	itsiLimiter = util.NewLimiter(concurrency)
}
