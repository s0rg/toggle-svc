package redis

import (
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mediocregopher/radix/v3"

	"github.com/s0rg/toggle-svc/pkg/toggle"
)

var errReplyNotFull = errors.New("reply not full")

type state struct {
	Segment string  `json:"key"`
	Toggles []int64 `json:"ids"`
}

type Store interface {
	ClientsInc(app, version, platform string) (int64, error)
	MarkAlive(string) error
	DropState(string) error
	GetState(string) ([]int64, bool, error)
	IsAlive(string) (bool, error)
	TogglesGet(app, version, platform string, keys toggle.Keys) ([]int64, error)
	TogglesIncr(app, version, platform string, keys toggle.Keys) (string, error)
}

type redis struct {
	c   radix.Client
	exp string
}

// New create new redis store.
func New(c radix.Client, d time.Duration) Store {
	return &redis{
		c:   c,
		exp: strconv.Itoa(int(d.Seconds())),
	}
}

// ClientsInc increases total number of clients in given segment.
func (r *redis) ClientsInc(app, version, platform string) (count int64, err error) {
	key := clientsKey(segmentKey(app, version, platform))
	err = r.c.Do(radix.Cmd(&count, "INCR", key))

	return
}

// MarkAlive updates key expire time.
func (r *redis) MarkAlive(key string) (err error) {
	return r.c.Do(radix.Cmd(nil, "EXPIRE", aliveKey(key), r.exp))
}

// IsAlive checks key for existence.
func (r *redis) IsAlive(key string) (yes bool, err error) {
	var rc int

	if err = r.c.Do(radix.Cmd(&rc, "EXISTS", aliveKey(key))); err != nil {
		return
	}

	return rc == 1, nil
}

// DropState cleans-up state and decrease counters.
func (r *redis) DropState(key string) (err error) {
	var (
		skey = stateKey(key)
		raw  string
		s    state
	)

	if err = r.c.Do(radix.Cmd(&raw, "GET", skey)); err != nil || raw == "" {
		return
	}

	if s, err = decodeState(raw); err != nil {
		return
	}

	if err = r.togglesDecr(s.Segment, s.Toggles); err != nil {
		return
	}

	return r.c.Do(radix.Cmd(nil, "DEL", skey))
}

// GetState returns toggles ids from state.
func (r *redis) GetState(key string) (ids []int64, found bool, err error) {
	var (
		raw string
		s   state
	)

	if found, err = r.IsAlive(key); err != nil || !found {
		return
	}

	if err = r.c.Do(radix.Cmd(&raw, "GET", stateKey(key))); err != nil || raw == "" {
		found = false

		return
	}

	if s, err = decodeState(raw); err != nil {
		return
	}

	return s.Toggles, true, nil
}

// TogglesIncr increase counters and save state (returning it id) for given segment and keys.
func (r *redis) TogglesIncr(app, version, platform string, keys toggle.Keys) (key string, err error) {
	key = uuid.New().String()

	s := state{
		Segment: segmentKey(app, version, platform),
	}

	err = r.c.Do(radix.WithConn(s.Segment, func(rc radix.Conn) (err error) {
		if err = rc.Do(radix.Cmd(nil, "MULTI")); err != nil {
			return
		}

		defer func() {
			if err != nil {
				_ = rc.Do(radix.Cmd(nil, "DISCARD"))
			}
		}()

		for i := 0; i < len(keys); i++ {
			if keys[i].Rate == 0 {
				continue
			}

			keyID := keys[i].ID
			if err = rc.Do(radix.Cmd(nil, "INCR", toggleKey(s.Segment, keyID))); err != nil {
				return
			}

			s.Toggles = append(s.Toggles, keyID)
		}

		var state string

		if state, err = encodeState(s); err != nil {
			return
		}

		if err = rc.Do(radix.Cmd(nil, "SET", stateKey(key), state)); err != nil {
			return
		}

		if err = rc.Do(radix.Cmd(nil, "SETEX", aliveKey(key), r.exp, "1")); err != nil {
			return
		}

		return rc.Do(radix.Cmd(nil, "EXEC"))
	}))

	return key, err
}

func (r *redis) togglesDecr(segment string, keysID []int64) (err error) {
	err = r.c.Do(radix.WithConn(segment, func(rc radix.Conn) (err error) {
		if err = rc.Do(radix.Cmd(nil, "MULTI")); err != nil {
			return
		}

		defer func() {
			if err != nil {
				_ = rc.Do(radix.Cmd(nil, "DISCARD"))
			}
		}()

		for i := 0; i < len(keysID); i++ {
			if err = rc.Do(radix.Cmd(nil, "DECR", toggleKey(segment, keysID[i]))); err != nil {
				return
			}
		}

		if err = rc.Do(radix.Cmd(nil, "DECR", clientsKey(segment))); err != nil {
			return
		}

		return rc.Do(radix.Cmd(nil, "EXEC"))
	}))

	return err
}

// TogglesGet returns slice of toggles counters.
func (r *redis) TogglesGet(app, version, platform string, keys toggle.Keys) (rv []int64, err error) {
	var result []string

	segment := segmentKey(app, version, platform)

	err = r.c.Do(radix.WithConn(segment, func(rc radix.Conn) (err error) {
		if err = rc.Do(radix.Cmd(nil, "MULTI")); err != nil {
			return
		}

		defer func() {
			if err != nil {
				_ = rc.Do(radix.Cmd(nil, "DISCARD"))
			}
		}()

		for i := 0; i < len(keys); i++ {
			key := toggleKey(segment, keys[i].ID)

			if err = rc.Do(radix.Cmd(nil, "GET", key)); err != nil {
				return err
			}
		}

		return rc.Do(radix.Cmd(&result, "EXEC"))
	}))
	if err != nil {
		return
	}

	if len(result) != len(keys) {
		err = errReplyNotFull

		return
	}

	rv = make([]int64, len(result))

	for i, r := range result {
		if r == "" {
			continue
		}

		if rv[i], err = strconv.ParseInt(r, 10, 64); err != nil {
			return
		}
	}

	return rv, err
}
