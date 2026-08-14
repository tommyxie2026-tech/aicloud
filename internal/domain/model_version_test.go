package domain

import (
	"reflect"
	"testing"
)

func TestModelVersionContainsNoRuntimeState(t *testing.T) {
	typ := reflect.TypeOf(ModelVersion{})
	for _, forbidden := range []string{
		"Provider",
		"Endpoint",
		"DeploymentMode",
		"Pricing",
		"Health",
		"HealthCheckedAt",
		"P95LatencyMS",
		"ErrorRate",
		"QuotaRemaining",
		"CapacityAvailable",
		"QueueDepth",
		"ServiceTiers",
		"InferenceEfforts",
		"DataResidency",
	} {
		if _, ok := typ.FieldByName(forbidden); ok {
			t.Fatalf("ModelVersion must not contain runtime field %s", forbidden)
		}
	}
}
