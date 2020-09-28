package toggle

type (
	// Key holds single toggle key params.
	Key struct {
		ID   int64   `json:"id"`
		Rate float64 `json:"rate"`
		Name string  `json:"key"`
	}

	// Keys is a shorthand for []Key.
	Keys []Key
)

func toggleRate(total, curr int64) float64 {
	// We need curr+1 cause total already counts us, but curr is not.
	return float64(curr+1) / float64(total)
}

// DisableByRate updates toggles states, switching off currently over-used toggles.
func (k Keys) DisableByRate(total int64, counts []int64) {
	for i := 0; i < len(k); i++ {
		pk, pc := &k[i], counts[i]

		if pk.Rate < 1.0 && toggleRate(total, pc) > pk.Rate {
			pk.Rate = 0
		}
	}
}

// Names returns slice of toggles keys names.
func (k Keys) Names() (rv []string) {
	for i := 0; i < len(k); i++ {
		if k[i].Rate > 0 {
			rv = append(rv, k[i].Name)
		}
	}

	return rv
}

// EnableByID enables keys by their id.
func (k Keys) EnableByID(ids []int64) {
	idSet := make(map[int64]struct{})

	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	for i := 0; i < len(k); i++ {
		pk := &k[i]

		pk.Rate = 0
		if _, ok := idSet[pk.ID]; ok {
			pk.Rate = 1
		}
	}
}
