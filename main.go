package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/ripienaar/ttnbridge/processors/graphite"
	"github.com/ripienaar/ttnbridge/processors/logger"
	"github.com/ripienaar/ttnbridge/receiver"
	"github.com/ripienaar/ttnbridge/ttn"
)

var (
	fatalReason  string
	log          *logrus.Entry
	listenerHost *string
	listenerPort *int
	listenKey    *string

	loggerProcessor *bool

	graphiteProcessor *bool
	graphiteHost      *string
	graphitePort      *int
	graphiteTries     *int
	graphitePrefix    *string

	Version = "development"
)

type Processor interface {
	Process(interface{}) error
	Name() string
}

func process(ctx context.Context, processors []Processor, work chan *ttn.WebhookIntegrationPush, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case push := <-work:
			for _, p := range processors {
				err := p.Process(push)
				if err != nil {
					log.Printf("Could not publish to %q processor: %s", p.Name(), err)
				}
			}

		case <-ctx.Done():
			log.Warn("Message processing shutting down")
			return
		}
	}
}

func registerProcessors(ctx context.Context, wg *sync.WaitGroup) ([]Processor, error) {
	processors := []Processor{}

	if *loggerProcessor {
		processors = append(processors, logger.New(ctx, wg, log))
	}

	if *graphiteProcessor {
		p, err := graphite.New(ctx, wg, graphiteHost, graphitePort, graphitePrefix, graphiteTries, log)
		if err != nil {
			return nil, err
		}
		processors = append(processors, p)
	}

	return processors, nil
}

func IsStdoutTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func setupLogger() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	if !IsStdoutTTY() {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
	log = logrus.NewEntry(logger)
}

func main() {
	setupLogger()

	app := kingpin.New("ttnbridge", "The Things Network HTTP Push bridge")
	app.Author("R.I.Pienaar <rip@devco.net>")
	app.Version(Version)

	listenerHost = app.Flag("listen-host", "Host to listen on").Default("0.0.0.0").Envar("LISTEN_HOST").String()
	listenerPort = app.Flag("listen-port", "Port to listen on").Default("80").Envar("LISTEN_PORT").Int()
	listenKey = app.Flag("listen-key", "Key to expect in the BridgeKey header").Required().Envar("LISTEN_KEY").String()
	loggerProcessor = app.Flag("logger", "Enable the Logger processor").Default("true").Envar("LOGGER_PROCESSOR").Bool()
	graphiteProcessor = app.Flag("graphite", "Enable the Graphite processor").Envar("GRAPHITE_PROCESSOR").Bool()
	graphiteHost = app.Flag("graphite-host", "Hostname of the Graphite server").Envar("GRAPHITE_HOST").String()
	graphitePort = app.Flag("graphite-port", "Port of the Graphite server").Envar("GRAPHITE_PORT").Int()
	graphiteTries = app.Flag("graphite-tries", "Number of times to retry publishing to graphite").Envar("GRAPHITE_TRIES").Default("10").Int()
	graphitePrefix = app.Flag("graphite-prefix", "Prefix used for the Graphite metrics").Envar("GRAPHITE_PREFIX").Default("ttn").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	work := make(chan *ttn.WebhookIntegrationPush, 1)
	listener, err := receiver.New(*listenerHost, *listenerPort, *listenKey, work, log)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	processors, err := registerProcessors(ctx, wg)
	if err != nil {
		log.Fatal(err)
	}
	if len(processors) == 0 {
		log.Fatalf("No processors are active")
	}

	wg.Add(1)
	go interruptWatcher(ctx, cancel, wg)

	wg.Add(1)
	go process(ctx, processors, work, wg)

	wg.Add(1)
	err = listener.Start(ctx, wg)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP listener failed: %s", err)
	}

	wg.Wait()

	log.Warn("Exiting cleanly after all routines completed")
	os.Exit(0)
}

func forcequit() {
	grace := 2 * time.Second

	<-time.NewTimer(grace).C

	log.Errorf("Forced shutdown triggered after %v", grace)

	os.Exit(1)
}

func interruptWatcher(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	defer wg.Done()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case sig := <-sigs:
			go forcequit()

			log.Warnf("Triggering shutdown on %s", sig)
			cancel()

		case <-ctx.Done():
			return
		}
	}
}
