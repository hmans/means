package collection

import (
	"math"
	"testing"
)

func TestSemanticThreshold(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	for _, title := range []string{"apple", "engine"} {
		if _, err := c.Create(map[string]any{"title": title}, ""); err != nil {
			t.Fatal(err)
		}
	}
	api.response = `{"data":[{"index":0,"embedding":[-1,0]}]}`
	if _, err := c.Create(map[string]any{"title": "opposite"}, ""); err != nil {
		t.Fatal(err)
	}
	api.response = ""
	zero, all, one := 0.0, -1.0, 1.0
	for _, tt := range []struct {
		name         string
		threshold    *float64
		limit, count int
		more         bool
	}{
		{"default", nil, 1, 1, false},
		{"zero inclusive", &zero, 0, 2, false},
		{"all scores", &all, 0, 3, false},
		{"limit after threshold", &zero, 1, 1, true},
		{"one inclusive", &one, 0, 1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, err := c.Search(SearchOptions{Query: "fruit", Semantic: true, MinScore: tt.threshold, Limit: tt.limit})
			if err != nil || len(r.Documents) != tt.count || r.HasMore != tt.more {
				t.Fatalf("result: %+v %v", r, err)
			}
		})
	}
	r, err := c.Search(SearchOptions{Query: "fruit", Semantic: true, Filter: map[string]any{"missing": true}})
	if err != nil || len(r.Documents) != 0 || r.HasMore {
		t.Fatalf("empty: %+v %v", r, err)
	}
}

func TestSemanticThresholdValidation(t *testing.T) {
	c := embeddingCollection(t)
	for _, value := range []float64{-1.1, 1.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := c.Search(SearchOptions{Semantic: true, Query: "fruit", MinScore: &value}); err == nil {
			t.Fatalf("accepted %v", value)
		}
	}
	zero := 0.0
	if _, err := c.Search(SearchOptions{Query: "fruit", MinScore: &zero}); err == nil {
		t.Fatal("accepted min_score without semantic")
	}
}
