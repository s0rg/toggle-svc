package redis

import (
	"encoding/base64"
	"hash/fnv"
	"io"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
)

const (
	keyPrefix  = "svc-toggle"
	keyClients = "clients"
	keyToggles = "toggles"
	keyCount   = "count"
	keyState   = "state"
	keyAlive   = "alive"
)

var b64enc = base64.RawURLEncoding

func segmentKey(app, version, platform string) string {
	h := fnv.New128a()
	_, _ = io.WriteString(h, strings.Join([]string{app, version, platform}, ":"))

	return b64enc.EncodeToString(h.Sum(nil))
}

func clientsKey(appKey string) string {
	return strings.Join([]string{keyPrefix, keyClients, appKey, keyCount}, ":")
}

func toggleKey(appKey string, toggleID int64) string {
	return strings.Join([]string{keyPrefix, keyToggles, appKey, strconv.Itoa(int(toggleID)), keyCount}, ":")
}

func stateKey(key string) string {
	return strings.Join([]string{keyPrefix, keyClients, key, keyState}, ":")
}

func aliveKey(key string) string {
	return strings.Join([]string{keyPrefix, keyClients, key, keyAlive}, ":")
}

func encodeState(s state) (rv string, err error) {
	var b []byte

	if b, err = cbor.Marshal(s); err != nil {
		return
	}

	rv = b64enc.EncodeToString(b)

	return rv, nil
}

func decodeState(s string) (rv state, err error) {
	var b []byte

	if b, err = b64enc.DecodeString(s); err != nil {
		return
	}

	err = cbor.Unmarshal(b, &rv)

	return rv, err
}
