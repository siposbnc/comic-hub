package providers

import (
	"math"
	"testing"
)

func TestScoreSeries(t *testing.T) {
	tests := []struct {
		name  string
		local LocalSeries
		cand  SeriesCandidate
		want  float64 // expected score, checked to 0.01
	}{
		{
			name:  "exact name + year",
			local: LocalSeries{Name: "Batman", Year: 2016},
			cand:  SeriesCandidate{Name: "Batman", Year: 2016},
			want:  1.0,
		},
		{
			name:  "exact name, year off by one",
			local: LocalSeries{Name: "Wonder Woman", Year: 2016},
			cand:  SeriesCandidate{Name: "Wonder Woman", Year: 2017},
			want:  (weightName*1 + weightYear*0.6) / (weightName + weightYear),
		},
		{
			name:  "qualifiers ignored in name, year+count present",
			local: LocalSeries{Name: "Batman", Year: 2016, IssueCount: 50},
			cand:  SeriesCandidate{Name: "Batman (2016)", Year: 2016, IssueCount: 50},
			want:  1.0,
		},
		{
			name:  "name only (no year/count) exact",
			local: LocalSeries{Name: "Saga"},
			cand:  SeriesCandidate{Name: "Saga", Year: 2012, IssueCount: 60},
			want:  1.0,
		},
		{
			name:  "different series scores zero",
			local: LocalSeries{Name: "Batman", Year: 2016},
			cand:  SeriesCandidate{Name: "Superman", Year: 2016},
			want:  weightYear / (weightName + weightYear), // name 0, year 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreSeries(tt.local, tt.cand)
			if math.Abs(got-tt.want) > 0.01 {
				t.Fatalf("ScoreSeries = %.3f, want %.3f", got, tt.want)
			}
		})
	}
}

func TestNameSimilarity(t *testing.T) {
	if s := nameSimilarity("The Batman", "Batman"); s != 1 {
		t.Errorf("leading article should be stripped: got %.3f, want 1", s)
	}
	if s := nameSimilarity("Batman", "Batman Rebirth"); math.Abs(s-2.0/3.0) > 0.01 {
		t.Errorf("subset name: got %.3f, want ~0.667", s)
	}
	if s := nameSimilarity("Batman", "Superman"); s != 0 {
		t.Errorf("no shared tokens: got %.3f, want 0", s)
	}
	if s := nameSimilarity("X-Men", "x men"); s != 1 {
		t.Errorf("punctuation normalized: got %.3f, want 1", s)
	}
}

func TestRankSeries(t *testing.T) {
	local := LocalSeries{Name: "Saga", Year: 2012, IssueCount: 60}
	cands := []SeriesCandidate{
		{ProviderID: "wrong", Name: "Sagas of the Sword", Year: 1990, IssueCount: 12},
		{ProviderID: "right", Name: "Saga", Year: 2012, IssueCount: 60},
		{ProviderID: "close", Name: "Saga", Year: 2013, IssueCount: 55},
	}

	ranked := RankSeries(local, cands)

	if ranked[0].ProviderID != "right" {
		t.Fatalf("best match = %q, want \"right\"", ranked[0].ProviderID)
	}
	if ranked[len(ranked)-1].ProviderID != "wrong" {
		t.Fatalf("worst match = %q, want \"wrong\"", ranked[len(ranked)-1].ProviderID)
	}
	for i := 1; i < len(ranked); i++ {
		if ranked[i-1].Score < ranked[i].Score {
			t.Fatalf("not sorted best-first at %d: %.3f < %.3f", i, ranked[i-1].Score, ranked[i].Score)
		}
	}
	// Ranking must not mutate the caller's slice.
	if cands[0].Score != 0 {
		t.Fatalf("input slice mutated: Score = %.3f", cands[0].Score)
	}
}
