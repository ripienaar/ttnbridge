package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ripienaar/ttnbridge/ttn"
)

type Receiver struct {
	port   int
	host   string
	key    string
	mux    *http.ServeMux
	outbox chan *ttn.WebhookIntegrationPush
	*logrus.Entry
}

func New(host string, port int, key string, q chan *ttn.WebhookIntegrationPush, logger *logrus.Entry) (*Receiver, error) {
	r := &Receiver{
		port:   port,
		host:   host,
		key:    key,
		outbox: q,
		Entry:  logger.WithField("component", "receiver"),
	}

	r.mux = http.NewServeMux()
	r.mux.HandleFunc("/v1/ttn", r.keyAuthMiddleware(http.HandlerFunc(r.handleTTN)))
	r.mux.HandleFunc("/health", r.handleAlive)

	return r, nil
}

func (r *Receiver) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	srv := &http.Server{Addr: fmt.Sprintf("%s:%d", r.host, r.port), Handler: r.mux}
	ec := make(chan error, 1)

	go func() {
		r.Infof("Listening on %s", srv.Addr)
		ec <- srv.ListenAndServe()
	}()

	select {
	case err := <-ec:
		return err
	case <-ctx.Done():
		r.Warn("Shutting down listening socket with 1 second timeout")
		shutdown, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		return srv.Shutdown(shutdown)
	}
}

func (r *Receiver) parseBody(req *http.Request, result interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	err = json.Unmarshal(body, result)
	if err != nil {
		return err
	}

	return nil
}

func (r *Receiver) keyAuthMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("BridgeKey") != r.key {
			w.WriteHeader(http.StatusUnauthorized)
			r.Warnf("Unauthorized %s request %s from %s", req.Method, req.URL, req.RemoteAddr)
			return
		}

		next.ServeHTTP(w, req)
	}
}
func (r *Receiver) handleAlive(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK\n"))
}

func (r *Receiver) handleTTN(w http.ResponseWriter, req *http.Request) {
	r.Infof("Handling %d byte %s from %s", req.ContentLength, req.Method, req.RemoteAddr)

	switch req.Method {
	case "POST":
		push := &ttn.WebhookIntegrationPush{}
		err := r.parseBody(req, push)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Could not parse request: %s", err)))
			return
		}

		r.outbox <- push

		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}
