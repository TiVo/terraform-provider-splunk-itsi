package config

import (
	"time"

	"github.com/lestrrat-go/backoff/v2"
	"github.com/tivo/terraform-provider-splunk-itsi/models"
)

type Config struct {
	// General Options
	Verbose     bool
	Concurrency int

	// Connection Options
	Host     string
	Port     int
	Insecure bool

	// Authentication Options
	AccessToken string
	User        string
	Password    string
}

func (c Config) ClientConfig() models.ClientConfig {
	return models.ClientConfig{
		BearerToken: c.AccessToken,
		Host:        c.Host,
		Port:        c.Port,
		User:        c.User,
		Password:    c.Password,
		SkipTLS:     c.Insecure,
		Concurrency: c.Concurrency,
		RetryPolicy: backoff.Exponential(
			backoff.WithMinInterval(500*time.Millisecond),
			backoff.WithMaxInterval(time.Minute),
			backoff.WithJitterFactor(0.05),
			backoff.WithMaxRetries(3),
		),
	}
}
