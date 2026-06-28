package domain

import "context"

// Smart-list rule fields and operators. The service validates against these; the store
// compiles them to SQL. Keep the two in lockstep when extending.
const (
	SmartFieldTag        = "tag"        // value = tag id
	SmartFieldSeries     = "series"     // value = series name
	SmartFieldPublisher  = "publisher"  // value = publisher name
	SmartFieldFormat     = "format"     // value = file format (cbz, cbr, …)
	SmartFieldAgeRating  = "ageRating"  // value = age rating label
	SmartFieldReadStatus = "readStatus" // value = read | unread | in_progress

	SmartOpIs       = "is"
	SmartOpIsNot    = "isNot"
	SmartOpContains = "contains"

	SmartMatchAll = "all"
	SmartMatchAny = "any"
)

// SmartRule is one predicate: a field, an operator, and a value.
type SmartRule struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

// SmartRules is the stored rule set: how to combine the rules ("all"=AND, "any"=OR).
type SmartRules struct {
	Match string      `json:"match"`
	Rules []SmartRule `json:"rules"`
}

// SmartList is a saved rule set whose contents are computed on demand.
type SmartList struct {
	ID        string
	OwnerID   string
	Name      string
	Rules     SmartRules
	BookCount int // evaluated per acting user on read; not stored
	CreatedAt int64
	UpdatedAt int64
}

// SmartListRepository persists smart lists and evaluates their rules against the catalog.
// Evaluation/Count take the acting user because read-status rules are per-user.
type SmartListRepository interface {
	Create(ctx context.Context, l SmartList) (SmartList, error)
	Get(ctx context.Context, id string) (SmartList, error)
	List(ctx context.Context) ([]SmartList, error)
	Update(ctx context.Context, l SmartList) error
	Delete(ctx context.Context, id string) error

	// Evaluate returns matching book ids (newest-added first) up to limit (<=0 = all).
	Evaluate(ctx context.Context, rules SmartRules, userID string, limit int) ([]string, error)
	// Count returns how many books match.
	Count(ctx context.Context, rules SmartRules, userID string) (int, error)
}
