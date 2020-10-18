package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/s0rg/toggle-svc/pkg/api"
	"github.com/s0rg/toggle-svc/pkg/db"
	"github.com/s0rg/toggle-svc/pkg/redis"
	"github.com/s0rg/toggle-svc/pkg/toggle"
)

const (
	waiterBufLen = 128
	waiterPeriod = time.Minute
	waiterWait   = time.Second / 2
)

var (
	errWaiterOverflow = errors.New("waiter overflow")
	errClientNotAlive = errors.New("not alive")
)

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
	h := api.New(s, s.db)

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

func (s *service) loadState(key string, keys toggle.Keys) (found bool, err error) {
	var keyIDs []int64

	if keyIDs, found, err = s.rd.GetState(key); err != nil || !found {
		return
	}

	if err = s.rd.MarkAlive(key); err != nil {
		return
	}

	keys.EnableByID(keyIDs)

	return
}

func (s *service) makeState(app, version, platform string, keys toggle.Keys) (key string, err error) {
	var (
		total  int64
		counts []int64
	)

	if counts, err = s.rd.TogglesGet(app, version, platform, keys); err != nil {
		return
	}

	if total, err = s.rd.ClientsInc(app, version, platform); err != nil {
		return
	}

	keys.DisableByRate(total, counts)

	if key, err = s.rd.TogglesIncr(app, version, platform, keys); err != nil {
		return
	}

	err = s.watchKey(key)

	return key, err
}

func (s *service) CodeToggles(
	ctx context.Context,
	app, version, platform, toggleID string,
) (
	clientID string,
	keys toggle.Keys,
	err error,
) {
	var (
		appID int64
		found bool
	)

	if appID, err = s.db.GetAppID(ctx, app); err != nil {
		return
	}

	if keys, err = s.db.GetAppFeatures(ctx, appID, version, platform); err != nil {
		return
	}

	if toggleID != "" {
		if found, err = s.loadState(toggleID, keys); err != nil {
			return
		}
	}

	clientID = toggleID

	if !found {
		clientID, err = s.makeState(app, version, platform, keys)
	}

	return clientID, keys, err
}

func (s *service) MarkAlive(_ context.Context, clientID string) (err error) {
	var alive bool

	if alive, err = s.rd.IsAlive(clientID); err != nil {
		return
	}

	if !alive {
		return errClientNotAlive
	}

	return s.rd.MarkAlive(clientID)
}
