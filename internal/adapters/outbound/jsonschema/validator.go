package jsonschemaadapter

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) Validate(schema json.RawMessage, candidate json.RawMessage) error {
	var candidateValue any
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.UseNumber()
	if err := decoder.Decode(&candidateValue); err != nil {
		return fmt.Errorf("decode candidate: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("request.json", bytes.NewReader(schema)); err != nil {
		return fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := compiler.Compile("request.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}
	if err := compiled.Validate(candidateValue); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}
