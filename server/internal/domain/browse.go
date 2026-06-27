package domain

// OwnerUserID is the id of the single implicit owner in embedded mode (seeded by
// migration 0002). Progress and other per-user data are attributed to it until auth
// mode introduces real accounts.
const OwnerUserID = "owner"

// SeriesSummary is a series plus the aggregates the library grid needs: how many books
// it has, how many the user has read, and which book supplies its cover.
type SeriesSummary struct {
	Series
	BookCount   int
	ReadCount   int
	CoverBookID string
}
