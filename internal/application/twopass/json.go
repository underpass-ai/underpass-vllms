package twopassapp

import (
	"encoding/json"
	"fmt"
	"strings"
)

func normalizeJSON(raw string) (json.RawMessage, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()

	var candidate any
	if err := decoder.Decode(&candidate); err != nil {
		return nil, fmt.Errorf("Pass 2 returned invalid JSON: %w", err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("Pass 2 returned multiple JSON values")
	}

	normalized, err := json.Marshal(candidate)
	if err != nil {
		return nil, fmt.Errorf("normalize JSON: %w", err)
	}

	return json.RawMessage(normalized), nil
}
