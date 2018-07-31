// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"time"
)

type CMQConfig struct {
	QueueName          string `config:"queuename"`
	URL                string `config:"url"`
	Region             string `config:"region"`
	SecretID           string `config:"secretid"`
	SecretKey          string `config:"secretkey"`
	PollingWaitSeconds int    `config:"pollingwaitseconds"`
}

type Config struct {
	Period time.Duration `config:"period"`
	CMQ    CMQConfig     `config:"cmq"`
}

var DefaultConfig = Config{
	Period: 1 * time.Second,
	// QueueName: "fran-test",
}
