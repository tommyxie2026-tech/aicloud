package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/model/provider"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

// JSONStructuredParser decodes strict JSON model output into schema types.
//
// The parser intentionally accepts JSON only in the MVP. YAML or markdown fenced
// blocks can be added later, but the first production parser should stay strict.
type JSONStructuredParser struct{}

func NewJSONStructuredParser() *JSONStructuredParser {
	return &JSONStructuredParser{}
}

func (p *JSONStructuredParser) Parse(schemaRef provider.OutputSchemaRef, raw string) (any, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return nil, fmt.Errorf("empty structured output")
	}
	if strings.HasPrefix(body, "```") {
		return nil, fmt.Errorf("markdown fenced output is not accepted; return raw JSON only")
	}

	switch schemaRef.Name {
	case schema.KindChangePlan:
		var out schema.ChangePlan
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindYamlPatchProposal:
		var out schema.YamlPatchProposal
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindRiskExplanation:
		var out schema.RiskExplanation
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindRollbackPlan:
		var out schema.RollbackPlan
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindValidationReport:
		var out schema.ValidationReport
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindStateSummary:
		var out schema.StateSummary
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	case schema.KindPolicyFailureExplanation:
		var out schema.PolicyFailureExplanation
		if err := decodeStrictJSON(body, &out); err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported output schema: %s", schemaRef.Name)
	}
}

func decodeStrictJSON(raw string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("unexpected trailing JSON content")
	}
	return nil
}
