package metricmock

import (
	"context"
	"testing"
	"time"

	"github.com/opsorch/opsorch-core/schema"
)

func TestQueryBuildsSeries(t *testing.T) {
	provAny, err := New(map[string]any{"source": "demo"})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	end := time.Now().UTC()
	start := end.Add(-10 * time.Minute)
	series, err := prov.Query(context.Background(), schema.MetricQuery{Expression: "latency_p99{service=\"checkout\"}", Start: start, End: end, Step: 60})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("expected series and baseline, got %d", len(series))
	}
	if series[0].Service != "checkout" {
		t.Fatalf("expected service field set, got %s", series[0].Service)
	}
	if series[0].Labels["service"] != "checkout" {
		t.Fatalf("expected service label set, got %+v", series[0].Labels)
	}
	if len(series[0].Points) == 0 {
		t.Fatalf("expected points in series")
	}
	if series[0].Metadata["source"] != "demo" {
		t.Fatalf("expected metadata source, got %v", series[0].Metadata["source"])
	}
}

func TestQueryDefaultsWindow(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) == 0 || len(series[0].Points) == 0 {
		t.Fatalf("expected default series with points")
	}
}

func TestQueryRespectsScope(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	scope := schema.QueryScope{Service: "svc-web", Environment: "staging", Team: "team-velocity"}
	series, err := prov.Query(context.Background(), schema.MetricQuery{Scope: scope})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if got := series[0].Labels["service"]; got != scope.Service {
		t.Fatalf("expected scope service label %s, got %v", scope.Service, got)
	}
	if got := series[0].Service; got != scope.Service {
		t.Fatalf("expected service field %s, got %s", scope.Service, got)
	}
	if got := series[0].Labels["env"]; got != scope.Environment {
		t.Fatalf("expected scope environment label %s, got %v", scope.Environment, got)
	}
	if got := series[0].Labels["team"]; got != scope.Team {
		t.Fatalf("expected scope team label %s, got %v", scope.Team, got)
	}
	if got, ok := series[0].Metadata["scope"].(schema.QueryScope); !ok || got != scope {
		t.Fatalf("expected scope metadata %+v, got %+v", scope, series[0].Metadata["scope"])
	}
}

func TestQueryKeepsPrimarySeriesValuesIntact(t *testing.T) {
	provAny, err := New(map[string]any{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	prov := provAny.(*Provider)

	series, err := prov.Query(context.Background(), schema.MetricQuery{Expression: "latency_p99"})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(series) < 2 {
		t.Fatalf("expected comparative series, got %d", len(series))
	}
	if len(series[0].Points) == 0 || len(series[1].Points) == 0 {
		t.Fatalf("expected both series to contain points")
	}
	if series[0].Points[0].Value == series[1].Points[0].Value {
		t.Fatalf("baseline modification should not change primary series values")
	}
}
