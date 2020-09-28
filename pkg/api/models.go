package api

type (
	key struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}

	reqAddToggles struct {
		App       string   `json:"app"`
		Version   string   `json:"version"`
		Platforms []string `json:"platforms"`
		Keys      []key    `json:"keys"`
	}

	reqEditToggle struct {
		App      string  `json:"app"`
		Version  string  `json:"version"`
		Platform string  `json:"platform"`
		Key      string  `json:"key"`
		Rate     float64 `json:"rate"`
	}

	reqAddApp struct {
		Apps []string `json:"apps"`
	}

	reqAlive struct {
		ID string `json:"id"`
	}

	reqGetToggles struct {
		App      string `json:"app"`
		Version  string `json:"version"`
		Platform string `json:"platform"`
	}

	respGetToggles struct {
		ID   string   `json:"id"`
		Keys []string `json:"keys"`
	}
)
