//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/db/migrations"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestPostgresCostEventUsesRouteTimePricingVersion(t *testing.T) {
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	schema := fmt.Sprintf("r6_cost_%d", time.Now().UnixNano())
	if _, err := db.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	defer func() { _, _ = db.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`) }()
	if _, err := db.ExecContext(ctx, `SET search_path TO `+schema); err != nil {
		t.Fatalf("set isolated search_path: %v", err)
	}
	if err := migrations.Run(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	routeAt := time.Now().UTC().Add(-10 * time.Minute)
	v2At := routeAt.Add(5 * time.Minute)

	if _, err := db.ExecContext(ctx, `INSERT INTO models(id,name,provider,version) VALUES ('model-r6','model-r6','provider-r6','v1')`); err != nil {
		t.Fatalf("insert model: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO model_deployments(id,model_id,model_version,provider,endpoint,deployment_mode,lifecycle_state,routing_eligible) VALUES ('dep-r6','model-r6','v1','provider-r6','https://example.invalid','hosted','ready',TRUE)`); err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO pricing_policies(id,version,deployment_id,currency,input_per_million,output_per_million,effective_from,effective_to,digest) VALUES ('price-r6','v1','dep-r6','USD',2,4,$1,$2,'digest-v1')`, routeAt.Add(-time.Hour), v2At); err != nil {
		t.Fatalf("insert pricing v1: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO tasks(id,input,status,trace_id,tenant_id,project_id,created_by,route_decision_id) VALUES ('task-r6','input','ROUTING','trace-r6','tenant-r6','project-r6','user-r6','')`); err != nil {
		t.Fatalf("insert task: %v", err)
	}

	selected := `{"modelId":"model-r6","modelVersion":"v1","deploymentId":"dep-r6","routeClass":"efficient","estimatedCost":0.006}`
	if _, err := db.ExecContext(ctx, `INSERT INTO route_decisions(id,task_id,selected,candidates,reason,fallback_chain,evidence_version,policy_version,created_at) VALUES ('route-r6','task-r6',$1::jsonb,$2::jsonb,'priced route','[]'::jsonb,'evidence-v1','policy-v1',$3)`, selected, `[`+selected+`]`, routeAt); err != nil {
		t.Fatalf("insert route decision: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET route_decision_id='route-r6' WHERE id='task-r6'`); err != nil {
		t.Fatalf("bind route decision: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO task_events(event_id,tenant_id,project_id,task_id,sequence,event_type,actor_principal_type,actor_subject_id,payload,trace_id,schema_version,occurred_at,created_at) VALUES ('event-r6','tenant-r6','project-r6','task-r6',1,'TaskRoutingStarted','user','user-r6',$1::jsonb,'trace-r6',1,$2,$2)`, `{"routeDecisionId":"route-r6","deploymentId":"dep-r6","estimatedInputTokens":1000,"estimatedOutputTokens":1000}`, routeAt); err != nil {
		t.Fatalf("insert routing event: %v", err)
	}

	var evidenceVersion, evidenceDigest string
	if err := db.QueryRowContext(ctx, `SELECT policy_version, policy_digest FROM route_pricing_evidence WHERE route_decision_id='route-r6' AND deployment_id='dep-r6'`).Scan(&evidenceVersion, &evidenceDigest); err != nil {
		t.Fatalf("read captured pricing evidence: %v", err)
	}
	if evidenceVersion != "v1" || evidenceDigest != "digest-v1" {
		t.Fatalf("captured evidence version=%q digest=%q want v1/digest-v1", evidenceVersion, evidenceDigest)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO pricing_policies(id,version,deployment_id,currency,input_per_million,output_per_million,effective_from,digest) VALUES ('price-r6','v2','dep-r6','USD',20,40,$1,'digest-v2')`, v2At); err != nil {
		t.Fatalf("insert pricing v2: %v", err)
	}

	repo := &PostgresDeploymentCostEvents{db: db}
	persisted, err := repo.Append(ctx, domain.CostEvent{
		ID: "cost-r6-input", TaskID: "task-r6", TraceID: "trace-r6",
		Component: domain.CostModelInput, Provider: "provider-r6", ModelID: "model-r6",
		ModelVersion: "v1", DeploymentID: "dep-r6", Quantity: 1000, Unit: "token",
		UnitPrice: 999, Amount: 999000, Currency: "USD", Attempt: 1, CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("append reconciled cost event: %v", err)
	}
	if persisted.Metadata["pricing_policy_version"] != "v1" || persisted.Metadata["pricing_policy_digest"] != "digest-v1" {
		t.Fatalf("cost metadata=%v want route-time v1 evidence", persisted.Metadata)
	}
	if math.Abs(persisted.UnitPrice-0.000002) > 1e-12 || math.Abs(persisted.Amount-0.002) > 1e-12 {
		t.Fatalf("reconciled unit_price=%v amount=%v want 0.000002/0.002", persisted.UnitPrice, persisted.Amount)
	}

	var storedVersion string
	var storedAmount float64
	if err := db.QueryRowContext(ctx, `SELECT metadata->>'pricing_policy_version', amount FROM cost_events WHERE id='cost-r6-input'`).Scan(&storedVersion, &storedAmount); err != nil {
		t.Fatalf("read stored cost event: %v", err)
	}
	if storedVersion != "v1" || math.Abs(storedAmount-0.002) > 1e-12 {
		t.Fatalf("stored version=%q amount=%v want v1/0.002", storedVersion, storedAmount)
	}
}
