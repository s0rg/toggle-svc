CREATE TABLE apps(
    id       BIGSERIAL    PRIMARY KEY,
    name     VARCHAR(255) NOT NULL,
    UNIQUE(name)
);

CREATE TABLE apps_versions(
    id         BIGSERIAL    PRIMARY KEY,
    app_id     BIGINT       NOT NULL,
    version    VARCHAR(64)  NOT NULL,
    platform   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, version, platform)
);

CREATE INDEX apps_versions_idx
    ON apps_versions (app_id, version, platform);

CREATE TABLE apps_features_keys(
    id         BIGSERIAL    PRIMARY KEY,
    app_id     BIGINT       NOT NULL,
    key        VARCHAR(255) NOT NULL,
    UNIQUE(app_id, key)
);

CREATE INDEX apps_features_keys_idx
    ON apps_features_keys (app_id);

CREATE TABLE apps_features_toggles(
    id         BIGSERIAL PRIMARY KEY,
    version_id BIGINT       NOT NULL,
    key_id     BIGINT       NOT NULL,
    rate       DECIMAL(3,2) NOT NULL,
    updated_at TIMESTAMP    NOT NULL DEFAULT NOW(),
    CHECK(rate >= 0 AND rate <= 1.0)
);

CREATE INDEX apps_features_toggles_idx
    ON apps_features_toggles (version_id, key_id);

