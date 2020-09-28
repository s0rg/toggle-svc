package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/s0rg/toggle-svc/pkg/api"
	"github.com/s0rg/toggle-svc/pkg/db"
	"github.com/s0rg/toggle-svc/pkg/redis"
)

const (
	waiterBufLen = 128
	waiterPeriod = time.Minute
	waiterWait   = time.Second / 2
)

var errWaiterOverflow = errors.New("waiter overflow")

type service struct {
	addr string
	db   db.Store
	rd   redis.Store
	wch  chan string
	qch  chan struct{}
}

func newService(addr string, dbs db.Store, rds redis.Store) *service {
	return &service{
		addr: addr,
		db:   dbs,
		rd:   rds,
		wch:  make(chan string, waiterBufLen),
		qch:  make(chan struct{}),
	}
}

func (s *service) dropIfDead(k string) (ok bool, err error) {
	if ok, err = s.rd.IsAlive(k); err != nil {
		return
	}

	if ok {
		return
	}

	err = s.rd.DropState(k)

	return ok, err
}

func (s *service) watcher() {
	state := make(map[string]struct{})

	t := time.NewTicker(waiterPeriod)
	defer t.Stop()

	defer close(s.wch)

	for {
		select {
		case key := <-s.wch:
			state[key] = struct{}{}
		case <-t.C:
			for k := range state {
				exists, err := s.dropIfDead(k)

				switch {
				case err != nil:
					log.Println("waiter: drop error:", err)

					fallthrough
				case exists:
					continue
				default:
					delete(state, k)
				}
			}
		case <-s.qch:
			return
		}
	}
}

func (s *service) watchKey(k string) (err error) {
	t := time.NewTimer(waiterWait)
	defer t.Stop()

	select {
	case s.wch <- k:
	case <-t.C:
		err = errWaiterOverflow
	}

	return err
}

func (s *service) Serve() (err error) {
	h := api.New(s.db, s.rd, s.watchKey)

	srv := &http.Server{
		Addr:           s.addr,
		Handler:        h.Mux(),
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go s.watcher()

	err = srv.ListenAndServe()

	close(s.qch)
	<-s.wch

	return err
}
