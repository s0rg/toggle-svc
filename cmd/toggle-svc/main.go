package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/mediocregopher/radix/v3"

	"github.com/s0rg/toggle-svc/pkg/app"
	appDB "github.com/s0rg/toggle-svc/pkg/app/db"
	"github.com/s0rg/toggle-svc/pkg/db"
	"github.com/s0rg/toggle-svc/pkg/redis"
	"github.com/s0rg/toggle-svc/pkg/retry"
)

const (
	appName       = "toggle-svc"
	maxRetries    = 3
	envKeysPrefix = "APP"
	envDBKey      = "DB"
	envAddr       = "ADDR"
	envRedisKey   = "REDIS"
	envExpiration = "EXPIRATION"
)

var (
	GitHash string
	BuildAt string
)

func run(app *app.App) (err error) {
	var (
		appAddr      = app.GetEnv(envAddr)
		appRedisDSN  = app.GetEnv(envRedisKey)
		appExpireStr = app.GetEnv(envExpiration)
		expireVal    time.Duration
		rdConn       radix.Client
		dbConn       *sql.DB
	)

	log.Println("build:", BuildAt, "starting")

	if expireVal, err = time.ParseDuration(appExpireStr); err != nil {
		return
	}

	steps := []retry.Step{
		{Name: "db", Do: func() (err error) {
			dbConn, err = appDB.ForApp(app, envDBKey)

			return
		}},
		{Name: "redis", Do: func() (err error) {
			rdConn, err = radix.Dial("tcp", appRedisDSN)

			return
		}},
	}

	if err := retry.RunSteps(maxRetries, steps); err != nil {
		return err
	}

	app.DeferClose(rdConn)

	s := newService(
		appAddr,
		db.New(dbConn),
		redis.New(rdConn, expireVal),
	)

	log.Println("serving on:", appAddr)

	return s.Serve()
}

func main() {
	app := app.New(appName).
		WithGitInfo(GitHash).
		WithEnvPrefix(envKeysPrefix).
		WithEnvKeys(envDBKey, envRedisKey, envExpiration, envAddr)

	if err := app.Init(); err != nil {
		log.Fatal(err)
	}

	defer app.Close()

	if err := run(app); err != nil {
		log.Println("app error:", err)
	}
}
