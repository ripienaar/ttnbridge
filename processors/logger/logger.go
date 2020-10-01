package logger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/ripienaar/ttnbridge/ttn"
)

type Processor struct {
	work chan interface{}
	sync.Mutex
	*logrus.Entry
}

func New(ctx context.Context, wg *sync.WaitGroup, log *logrus.Entry) *Processor {
	l := &Processor{
		work:  make(chan interface{}, 100),
		Entry: log.WithField("output", "logger"),
	}

	l.Infof("Enabling logger processor")

	wg.Add(1)
	go l.consume(ctx, wg)

	return l
}

func (l *Processor) consume(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case item := <-l.work:
			push, ok := item.(*ttn.WebhookIntegrationPush)
			if ok {
				log := l.Entry.
					WithField("app", push.AppID).
					WithField("device", push.DeviceID).
					WithField("created", push.Metadata.Time.Format(time.RFC3339))

				res := gjson.ParseBytes(push.PayloadFields)
				res.ForEach(func(key gjson.Result, value gjson.Result) bool {
					log.Infof("%s: %s", key.String(), value.String())
					return true
				})

			}
		case <-ctx.Done():
			l.Warnf("Shutting down logger processor")
			return
		}
	}
}

func (g *Processor) Name() string { return "logger" }
func (l *Processor) Process(item interface{}) error {
	l.Lock()
	defer l.Unlock()

	if len(l.work) == cap(l.work) {
		return fmt.Errorf("work queue is full")
	}

	l.work <- item

	return nil
}
