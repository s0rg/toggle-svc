package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/s0rg/toggle-svc/pkg/db"
	"github.com/s0rg/toggle-svc/pkg/redis"
	"github.com/s0rg/toggle-svc/pkg/toggle"
)

var errBadRequest = errors.New("bad request")

const (
	headerToggleID = "X-CodeToggleID"
)

type Muxer interface {
	Mux() http.Handler
}

type handlers struct {
	db       db.Store
	rd       redis.Store
	watchKey func(string) error
}

// New creates new api handlers.
func New(dbs db.Store, rds redis.Store, watcher func(string) error) Muxer {
	return &handlers{db: dbs, rd: rds, watchKey: watcher}
}

// Mux constructs new http.Handler for api.
func (h *handlers) Mux() http.Handler {
	var m http.ServeMux

	m.HandleFunc("/client/code-toggles", wrapAPI("client-get-toggles", h.GetCodeToggles))
	m.HandleFunc("/client/alive", wrapAPI("client-alive", h.Alive))

	m.HandleFunc("/apps", wrapAPI("apps-get", h.GetApps))
	m.HandleFunc("/apps/add", wrapAPI("apps-add", h.AddApps))

	m.HandleFunc("/toggles/add", wrapAPI("toggles-add", h.AddCodeToggles))
	m.HandleFunc("/toggles/edit", wrapAPI("toggles-edit", h.EditCodeToggles))

	return &m
}

func (h *handlers) loadState(key string, keys toggle.Keys) (rv respGetToggles, found bool, err error) {
	var keyIDs []int64

	if keyIDs, found, err = h.rd.GetState(key); err != nil || !found {
		return
	}

	if err = h.rd.MarkAlive(key); err != nil {
		return
	}

	keys.EnableByID(keyIDs)

	rv.ID = key

	return
}

func (h *handlers) makeState(app, version, platform string, keys toggle.Keys) (rv respGetToggles, err error) {
	var (
		total  int64
		counts []int64
	)

	if counts, err = h.rd.TogglesGet(app, version, platform, keys); err != nil {
		return
	}

	if total, err = h.rd.ClientsInc(app, version, platform); err != nil {
		return
	}

	keys.DisableByRate(total, counts)

	if rv.ID, err = h.rd.TogglesIncr(app, version, platform, keys); err != nil {
		return
	}

	err = h.watchKey(rv.ID)

	return rv, err
}

// GetCodeToggles returns enabled code toggles for client.
func (h *handlers) GetCodeToggles(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var req reqGetToggles

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	var (
		appID  int64
		loaded bool
		keys   toggle.Keys
		resp   respGetToggles
	)

	if appID, err = h.db.GetAppID(ctx, req.App); err != nil {
		return errBadRequest
	}

	if keys, err = h.db.GetAppFeatures(ctx, appID, req.Version, req.Platform); err != nil {
		return
	}

	if toggleID := r.Header.Get(headerToggleID); toggleID != "" {
		if resp, loaded, err = h.loadState(toggleID, keys); err != nil {
			return
		}
	}

	if !loaded {
		if resp, err = h.makeState(req.App, req.Version, req.Platform, keys); err != nil {
			return
		}
	}

	resp.Keys = keys.Names()

	return json.NewEncoder(w).Encode(&resp)
}

// AddCodeToggles adds new toggles for app.
func (h *handlers) AddCodeToggles(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var req reqAddToggles

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	var appID int64

	if appID, err = h.db.GetAppID(ctx, req.App); err != nil {
		return errBadRequest
	}

	keys := make(toggle.Keys, len(req.Keys))

	for i := 0; i < len(req.Keys); i++ {
		k, rk := &keys[i], &req.Keys[i]

		k.Name = rk.Name
		if rk.Enabled {
			k.Rate = 1.0
		}
	}

	return h.db.AddAppFeatures(ctx, appID, req.Version, req.Platforms, keys)
}

// AddApps adds new apps.
func (h *handlers) AddApps(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var req reqAddApp

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	if len(req.Apps) == 0 {
		return errBadRequest
	}

	return h.db.AddApps(ctx, req.Apps)
}

// GetApps returns slice of app names.
func (h *handlers) GetApps(ctx context.Context, w io.Writer, _ *http.Request) (err error) {
	var apps []string

	if apps, err = h.db.GetApps(ctx); err != nil {
		return
	}

	return json.NewEncoder(w).Encode(apps)
}

// Alive marks ToggleID as alive.
func (h *handlers) Alive(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var req reqAlive

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	var alive bool

	if alive, err = h.rd.IsAlive(req.ID); err != nil {
		return
	}

	if !alive {
		return errBadRequest
	}

	return h.rd.MarkAlive(req.ID)
}

// EditCodeToggles allows to edit toggle rate for specified key.
func (h *handlers) EditCodeToggles(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var (
		appID int64
		req   reqEditToggle
	)

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	if appID, err = h.db.GetAppID(ctx, req.App); err != nil {
		return errBadRequest
	}

	return h.db.EditAppFeature(ctx, appID, req.Version, req.Platform, req.Key, req.Rate)
}
