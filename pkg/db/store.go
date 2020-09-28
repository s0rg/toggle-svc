package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/s0rg/toggle-svc/pkg/toggle"
)

type store struct {
	db *sql.DB
}

type Store interface {
	GetApps(context.Context) ([]string, error)
	GetAppID(context.Context, string) (int64, error)
	GetAppFeatures(context.Context, int64, string, string) (toggle.Keys, error)
	AddApps(context.Context, []string) error
	AddAppFeatures(context.Context, int64, string, []string, toggle.Keys) error
	EditAppFeature(context.Context, int64, string, string, string, float64) error
}

// New create new DB store.
func New(db *sql.DB) Store {
	return &store{db: db}
}

// GetAppID returns id for given app name.
func (s *store) GetAppID(
	ctx context.Context,
	app string,
) (id int64, err error) {
	const query = `SELECT id FROM apps WHERE name = $1 LIMIT 1`

	err = s.db.QueryRowContext(ctx, query, strings.ToLower(app)).Scan(&id)

	return
}

// GetApps returns slice of available app names.
func (s *store) GetApps(
	ctx context.Context,
) (rv []string, err error) {
	const query = `SELECT name FROM apps`

	var rows *sql.Rows

	if rows, err = s.db.QueryContext(ctx, query); err != nil {
		return
	}

	defer rows.Close()

	var n string

	for rows.Next() {
		if err = rows.Scan(&n); err != nil {
			return
		}

		rv = append(rv, n)
	}

	return rv, rows.Err()
}

// GetAppFeatures returns slice of toggled features for given params.
func (s *store) GetAppFeatures(
	ctx context.Context,
	appID int64,
	version, platform string,
) (rv toggle.Keys, err error) {
	const query = `
SELECT
	t.id, k.key, t.rate
FROM
	apps_versions v
JOIN
	apps_features_keys k ON
		k.app_id = v.app_id
JOIN
	apps_features_toggles t ON
		t.version_id = v.id
		AND
		t.key_id = k.id
WHERE
	v.app_id = $1
	AND
	v.version = $2
	AND
	v.platform = $3
	AND
	t.rate > 0
`

	var rows *sql.Rows

	if rows, err = s.db.QueryContext(ctx, query, appID, version, platform); err != nil {
		return
	}

	defer rows.Close()

	var k toggle.Key

	for rows.Next() {
		if err = rows.Scan(&k.ID, &k.Name, &k.Rate); err != nil {
			return
		}

		rv = append(rv, k)
	}

	return rv, rows.Err()
}

// AddApps adds new app names.
func (s *store) AddApps(
	ctx context.Context,
	apps []string,
) error {
	const queryHead = `INSERT INTO apps(name) VALUES `

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	queryParts := make([]string, len(apps))
	args := make([]interface{}, len(apps))

	for i, a := range apps {
		queryParts[i] = fmt.Sprintf("($%d)", i+1)
		args[i] = strings.ToLower(a)
	}

	query := queryHead + strings.Join(queryParts, ",")

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *store) getOrCreateKeys(
	ctx context.Context,
	tx *sql.Tx,
	appID int64,
	keys toggle.Keys,
) (rv map[string]int64, err error) {
	const query = `
WITH new_app_key AS (
	INSERT INTO	apps_features_keys
		(app_id, key)
	VALUES
		($1, $2)
	ON CONFLICT DO NOTHING
	RETURNING id
)

SELECT id FROM new_app_key
UNION
SELECT id FROM apps_features_keys
WHERE
	app_id = $1 AND key = $2
LIMIT 1
`

	rv = make(map[string]int64)

	var id int64

	for i := 0; i < len(keys); i++ {
		k := keys[i].Name

		if err = tx.QueryRowContext(ctx, query, appID, k).Scan(&id); err != nil {
			return
		}

		rv[k] = id
	}

	return rv, nil
}

// AddAppFeatures adds new version, platforms and toggles for given app.
func (s *store) AddAppFeatures(
	ctx context.Context,
	appID int64,
	version string,
	platforms []string,
	keys toggle.Keys,
) error {
	const (
		addVersion = `
INSERT INTO apps_versions
	(app_id, version, platform)
VALUES
	($1, $2, $3)
RETURNING id`

		addToggle = `
INSERT INTO apps_features_toggles
	(version_id, key_id, rate)
VALUES
	($1, $2, $3)
`
	)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	appKeys, err := s.getOrCreateKeys(ctx, tx, appID, keys)
	if err != nil {
		return err
	}

	var versionID int64

	for i := 0; i < len(platforms); i++ {
		if err = tx.QueryRowContext(
			ctx, addVersion, appID, version, platforms[i],
		).Scan(
			&versionID,
		); err != nil {
			return err
		}

		for j := 0; j < len(keys); j++ {
			k := &keys[j]

			if _, err = tx.ExecContext(
				ctx, addToggle, versionID, appKeys[k.Name], k.Rate,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// EditAppFeature modifies rate for selected key.
func (s *store) EditAppFeature(
	ctx context.Context,
	appID int64,
	version string,
	platform string,
	key string,
	rate float64,
) (err error) {
	const (
		getToggleID = `
SELECT
	t.id
FROM
	apps_versions v
JOIN
	apps_features_keys k ON
		k.app_id = v.app_id
JOIN
	apps_features_toggles t ON
		t.version_id = v.id
		AND
		t.key_id = k.id
WHERE
	v.app_id = $1
	AND
	v.version = $2
	AND
	v.platform = $3
	AND
	k.key = $4
LIMIT 1
`

		setRate = `
UPDATE apps_features_toggles
SET rate = $2, updated_at = NOW()
WHERE id = $1`
	)

	var toggleID int64

	if err = s.db.QueryRowContext(
		ctx, getToggleID, appID, version, platform, key,
	).Scan(&toggleID); err != nil {
		return
	}

	_, err = s.db.ExecContext(ctx, setRate, toggleID, rate)

	return err
}
