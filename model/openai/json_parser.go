package openai

import (
	"encoding/json"
	"fmt"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

// JSONParser provides a provider-neutral baseline parser. Domain-specific
// schema validation remains a separate validation step after parsing.
type JSONParser struct{}

func (JSONParser) Parse(schemaRef provider.OutputSchemaRef, raw string) (any, error) {
	if schemaRef.Name == "" {
		return nil, fmt.Errorf("output schema name is required")
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, fmt.Errorf("parse structured JSON for %s: %w", schemaRef.Name, err)
	}
	return value, nil
}
