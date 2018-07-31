package beater

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/yyff/cmqbeat/cmq"
	"github.com/yyff/cmqbeat/config"
)

// Cmqbeat configuration.
type Cmqbeat struct {
	done   chan struct{}
	config *config.Config
	client beat.Client
	cmqapi *cmq.CMQAPI
}

// New creates an instance of cmqbeat.
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	c := &config.DefaultConfig
	if err := cfg.Unpack(c); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}
	logp.Debug("", "Get cmqbeat config: ", c)
	api := cmq.NewCMQAPI(&c.CMQ)

	bt := &Cmqbeat{
		done:   make(chan struct{}),
		config: c,
		cmqapi: api,
	}
	return bt, nil
}

// Run starts cmqbeat.
func (bt *Cmqbeat) Run(b *beat.Beat) error {
	logp.Info("cmqbeat is running! Hit CTRL-C to stop it.")

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(bt.config.Period)
	api := bt.cmqapi
	counter := 1
	for {
		select {
		case <-bt.done:
			return nil
		case <-ticker.C:
		}
		msg, handle, err := api.RecvMsg()
		if err != nil {
			logp.Err("recv msg from queue:[%v] error: %v", bt.config.CMQ.QueueName, err)
			continue
		}
		if msg == "" {
			logp.Info("no msgs received")
			continue
		}

		event := beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"type":    b.Info.Name,
				"counter": counter,
				"message": msg,
			},
		}
		bt.client.Publish(event)
		err = api.DeleteMsg(handle)
		if err != nil {
			logp.Err("DeleteMsg error: ", err)
		}
		logp.Info("Event sent")
		counter++
	}
}

// Stop stops cmqbeat.
func (bt *Cmqbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
