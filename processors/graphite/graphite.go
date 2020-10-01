package graphite

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/ripienaar/ttnbridge/ttn"
)

type Processor struct {
	host   string
	port   int
	prefix string
	work   chan interface{}
	conn   net.Conn
	ctr    int
	tries  int

	sync.Mutex
	*logrus.Entry
}

func New(ctx context.Context, wg *sync.WaitGroup, host *string, port *int, prefix *string, tries *int, log *logrus.Entry) (*Processor, error) {
	if host == nil {
		return nil, fmt.Errorf("graphite processor requires a host")
	}
	if port == nil || *port == 0 {
		return nil, fmt.Errorf("graphite processor requires a port")
	}
	if prefix == nil || *prefix == "" {
		return nil, fmt.Errorf("graphite processor requires a prefix")
	}
	if tries == nil || *tries == 0 {
		return nil, fmt.Errorf("graphite tries can not be 0")
	}

	g := &Processor{
		host:   *host,
		port:   *port,
		prefix: *prefix,
		tries:  *tries,
		work:   make(chan interface{}, 100),
		Entry:  log.WithField("output", "graphite"),
	}

	g.Infof("Enabling graphite processor on %s:%d with prefix %s", g.host, g.port, g.prefix)

	wg.Add(1)
	go g.consume(ctx, wg)

	return g, nil
}

func (g *Processor) consume(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case item := <-g.work:
			push, ok := item.(*ttn.WebhookIntegrationPush)
			if ok {
				err := g.publishTTN(push)
				if err != nil {
					g.Logger.Errorf("could not publish item: %s", err)
				}
			}
		case <-ctx.Done():
			g.Warn("Shutting down graphite processor")
			g.close()

			return
		}
	}
}

func (g *Processor) publishTTN(push *ttn.WebhookIntegrationPush) (err error) {
	g.Lock()
	defer g.Unlock()

	tries := 0
	start := time.Now()
	for {
		tries++

		log := g.WithField("try", tries).WithField("app", push.AppID).WithField("device", push.DeviceID)

		if tries > g.tries {
			return fmt.Errorf("publishing message failed after %d tries, discarding", tries-1)
		}

		if g.conn == nil {
			err = g.reconnect()
			if err != nil {
				// todo: backoff
				log.Errorf("reconnecting to graphite failed: %s", err)
				time.Sleep(time.Second)
				continue
			}
		}

		err = g.conn.SetWriteDeadline(time.Now().Add(time.Second))
		if err != nil {
			g.reset()
			log.Errorf("could not set write deadline: %s", err)
			continue
		}

		ts := push.Metadata.Time.Unix()

		_, err = g.conn.Write([]byte(fmt.Sprintf("%s.%s.%s.seen 1 %d\n", g.prefix, push.AppID, push.DeviceID, ts)))
		if err != nil {
			g.reset()
			log.Errorf("could not write metric: %s", err)
			continue
		}

		res := gjson.ParseBytes(push.PayloadFields)
		res.ForEach(func(key gjson.Result, value gjson.Result) bool {
			g.Infof("key: %s value: %v (%s)", key.String(), value.String(), value.Type)
			if value.Type != gjson.Number {
				return true
			}

			if !key.Exists() {
				return true
			}

			ks := strings.Replace(key.String(), " ", "_", -1)

			_, err = g.conn.Write([]byte(fmt.Sprintf("%s.%s.%s.%s %s %d\n", g.prefix, push.AppID, push.DeviceID, ks, value.String(), ts)))
			if err != nil {
				g.reset()
				log.Errorf("could not write metric: %s", err)
				return false
			}

			return true
		})

		log.Infof("Handled push in %v", time.Now().Sub(start))
		g.ctr++
		return nil
	}
}

func (g *Processor) reset() {
	g.conn.Close()
	g.conn = nil
}

func (g *Processor) reconnect() (err error) {
	if g.conn != nil {
		g.reset()
	}

	g.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", g.host, g.port))

	return err
}

func (g *Processor) close() {
	g.Lock()
	defer g.Unlock()

	if g.conn != nil {
		g.conn.Close()
	}
}

func (g *Processor) Name() string { return "graphite" }
func (g *Processor) Process(item interface{}) error {
	g.Lock()
	defer g.Unlock()

	if len(g.work) == cap(g.work) {
		return fmt.Errorf("work queue is full")
	}

	g.work <- item

	return nil
}
