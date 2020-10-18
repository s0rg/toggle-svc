package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/s0rg/toggle-svc/pkg/toggle"
)

var errBadRequest = errors.New("bad request")

const (
	headerToggleID = "X-CodeToggleID"
)

type Muxer interface {
	Mux() http.Handler
}

type service interface {
	CodeToggles(ctx context.Context, app, version, platform, clientID string) (string, toggle.Keys, error)
	MarkAlive(ctx context.Context, clientID string) error
}

type store interface {
	AddApps(context.Context, []string) error
	GetApps(context.Context) ([]string, error)
	GetAppID(context.Context, string) (int64, error)
	AddAppFeatures(context.Context, int64, string, []string, toggle.Keys) error
	EditAppFeature(context.Context, int64, string, string, string, float64) error
}

type handlers struct {
	srv service
	db  store
}

// New creates new api handlers.
func New(
	srv service,
	db store,
) Muxer {
	return &handlers{srv: srv, db: db}
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

// GetCodeToggles returns enabled code toggles for client.
func (h *handlers) GetCodeToggles(ctx context.Context, w io.Writer, r *http.Request) (err error) {
	var req reqGetToggles

	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errBadRequest
	}

	var (
		resp respGetToggles
		keys toggle.Keys
	)

	toggleID := r.Header.Get(headerToggleID)

	if resp.ID, keys, err = h.srv.CodeToggles(ctx, req.App, req.Platform, req.Version, toggleID); err != nil {
		return
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
		return
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

	return h.srv.MarkAlive(ctx, req.ID)
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
