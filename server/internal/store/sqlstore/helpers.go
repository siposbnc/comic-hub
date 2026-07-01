package sqlstore

import "database/sql"

// nullString returns a NULL-able driver value: nil for the empty string, else s. The
// catalog stores absent optional text (descriptions, publishers, …) as SQL NULL.
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullInt returns nil for a zero value, else n. Used for optional integer columns
// (year, release_date, volume) where 0 means "unknown".
func nullInt(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

// nullFloat returns nil for a zero value, else f. Used for optional real columns
// (sort_number) where 0 means "unknown".
func nullFloat(f float64) any {
	if f == 0 {
		return nil
	}
	return f
}

// str unwraps a sql.NullString to its value (empty when NULL).
func str(n sql.NullString) string { return n.String }

// i64 unwraps a sql.NullInt64 to its value (0 when NULL).
func i64(n sql.NullInt64) int64 { return n.Int64 }

// f64 unwraps a sql.NullFloat64 to its value (0 when NULL).
func f64(n sql.NullFloat64) float64 { return n.Float64 }

// boolToInt maps a bool to the 0/1 integer SQLite stores.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
