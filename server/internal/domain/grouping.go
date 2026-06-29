package domain

// StoryArc is a provider-sourced narrative arc spanning several of a series' issues.
type StoryArc struct {
	ID          string
	SeriesID    string
	Name        string
	Description string
	IssueCount  int
}

// StoryArcInput is one arc to (re)write for a series, with its member book ids in issue
// order. ProviderID is the external (e.g. Comic Vine) arc id, used to de-dupe across issues.
type StoryArcInput struct {
	ProviderID  string
	Name        string
	Description string
	BookIDs     []string
}
