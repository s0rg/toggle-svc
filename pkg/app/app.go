//nolint:gocritic,goerr113
package app

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rs/zerolog"
)

// App base, nothing interesting.
type App struct {
	// Name is a shortcut for `name` argument.
	Name      string
	id        string
	git       string
	envPrefix string
	envKeys   []string
	closers   []io.Closer
	env       map[string]string
}

// New creates empty application with given name.
func New(name string) *App {
	return &App{Name: name}
}

// WithGitInfo sets 'git' tag in logger.
func (app *App) WithGitInfo(git string) *App {
	app.git = git

	return app
}

// WithEnvPrefix sets env vars common prefix.
func (app *App) WithEnvPrefix(prefix string) *App {
	app.envPrefix = prefix

	return app
}

// WithEnvKeys adds given keys as dependencies, they will be requested
// from env as-is (no uppercase conversion), also you can add prefix
// with WithEnvPrefix(), then {prefix}_{key} format will be used.
func (app *App) WithEnvKeys(keys ...string) *App {
	app.envKeys = append(app.envKeys, keys...)

	return app
}

// ID returns string in {name}/{pid}@{host} format, where:
// name - app name (see app.New)
// pid  - process id
// host - hostname.
func (app *App) ID() string {
	return app.id
}

func (app *App) setUpEnv() (err error) {
	var (
		envKey, envVal string
		keyFor         func(string) string
	)

	if app.envPrefix == "" {
		keyFor = func(s string) string {
			return s
		}
	} else {
		keyFor = func(s string) string {
			return app.envPrefix + "_" + s
		}
	}

	app.env = make(map[string]string)

	for _, k := range app.envKeys {
		envKey = keyFor(k)

		if envVal = os.Getenv(envKey); envVal == "" {
			return fmt.Errorf("app.env: %s key '%s' is not set or empty", k, envKey)
		}

		app.env[k] = envVal
	}

	return nil
}

// Init bootstraps new app:
//
// - Obtains keys, listed with WithEnvKeys(), from env, if some missing or empty - produces error
// - Setup app-level json logger (zerolog) as default (and transparent replacement for stdlib log)
//and fills name, hostname, pid (and optionally git) tags
// - (optionally) Setup app-level opentracing agent.
func (app *App) Init() (err error) {
	var host string

	if host, err = os.Hostname(); err != nil {
		return
	}

	if err = app.setUpEnv(); err != nil {
		return
	}

	pid := os.Getpid()

	app.id = fmt.Sprintf("%s/%d@%s", app.Name, pid, host)

	zctx := zerolog.New(os.Stderr).With().
		Timestamp().
		Str("name", app.Name).
		Str("hostname", host).
		Int("pid", pid)

	if app.git != "" {
		zctx = zctx.Str("git", app.git)
	}

	log.SetFlags(0)
	log.SetOutput(zctx.Logger())

	return err
}

// GetEnv get dependency, if key was not listed via WithEnvKeys() - does log.Fatal.
func (app *App) GetEnv(key string) string {
	val, ok := app.env[key]
	if !ok {
		log.Fatal("app: env key missing:", key)
	}

	return val
}

// DeferClose defer some closers, see app.Close.
func (app *App) DeferClose(c io.Closer) {
	app.closers = append(app.closers, c)
}

// Close - Closes all `io.Closer`, that was deffered by `app.DeferClose`
// if closer fails with error, it will be logged, and closing process proceed.
func (app *App) Close() {
	for _, c := range app.closers {
		if err := c.Close(); err != nil {
			log.Println("app: close error:", err)
		}
	}
}
