package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/siposbnc/comic-hub/server/internal/domain"
)

// smartListRepo persists smart lists and compiles their rule sets to SQL against book +
// series (+ read_progress for the acting user, + book_tag for tag rules).
type smartListRepo struct{ db *sql.DB }

func (r *smartListRepo) Create(ctx context.Context, l domain.SmartList) (domain.SmartList, error) {
	rules, err := json.Marshal(l.Rules)
	if err != nil {
		return domain.SmartList{}, err
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO smart_list (id, owner_id, name, rules, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		l.ID, nullString(l.OwnerID), l.Name, string(rules), l.CreatedAt, l.UpdatedAt)
	if err != nil {
		return domain.SmartList{}, err
	}
	return l, nil
}

func (r *smartListRepo) Get(ctx context.Context, id string) (domain.SmartList, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, owner_id, name, rules, created_at, updated_at FROM smart_list WHERE id = ?`, id)
	l, err := scanSmartList(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SmartList{}, domain.ErrNotFound
	}
	return l, err
}

func (r *smartListRepo) List(ctx context.Context) ([]domain.SmartList, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, rules, created_at, updated_at FROM smart_list ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.SmartList
	for rows.Next() {
		l, err := scanSmartList(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *smartListRepo) Update(ctx context.Context, l domain.SmartList) error {
	rules, err := json.Marshal(l.Rules)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE smart_list SET name = ?, rules = ?, updated_at = ? WHERE id = ?`,
		l.Name, string(rules), l.UpdatedAt, l.ID)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *smartListRepo) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM smart_list WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

func (r *smartListRepo) Evaluate(ctx context.Context, rules domain.SmartRules, userID string, limit int) ([]string, error) {
	where, args, err := compileSmartRules(rules, userID)
	if err != nil {
		return nil, err
	}
	q := `SELECT b.id FROM book b JOIN series s ON s.id = b.series_id
		WHERE ` + where + ` ORDER BY b.added_at DESC`
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *smartListRepo) Count(ctx context.Context, rules domain.SmartRules, userID string) (int, error) {
	where, args, err := compileSmartRules(rules, userID)
	if err != nil {
		return 0, err
	}
	var n int
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM book b JOIN series s ON s.id = b.series_id WHERE `+where, args...,
	).Scan(&n)
	return n, err
}

// compileSmartRules turns a rule set into a SQL boolean expression over aliases b (book)
// and s (series), plus its bound args. An empty rule set matches nothing.
func compileSmartRules(rules domain.SmartRules, userID string) (string, []any, error) {
	if len(rules.Rules) == 0 {
		return "0", nil, nil
	}
	joiner := " AND "
	if strings.EqualFold(rules.Match, domain.SmartMatchAny) {
		joiner = " OR "
	}
	preds := make([]string, 0, len(rules.Rules))
	var args []any
	for _, rule := range rules.Rules {
		p, a, err := compileSmartRule(rule, userID)
		if err != nil {
			return "", nil, err
		}
		preds = append(preds, p)
		args = append(args, a...)
	}
	return "(" + strings.Join(preds, joiner) + ")", args, nil
}

func compileSmartRule(rule domain.SmartRule, userID string) (string, []any, error) {
	negate := rule.Op == domain.SmartOpIsNot
	val := strings.TrimSpace(rule.Value)

	switch rule.Field {
	case domain.SmartFieldTag:
		if val == "" {
			return "", nil, fmt.Errorf("%w: tag rule needs a value", domain.ErrValidation)
		}
		return maybeNot(negate,
			`EXISTS (SELECT 1 FROM book_tag bt WHERE bt.book_id = b.id AND bt.tag_id = ?)`), []any{val}, nil

	case domain.SmartFieldSeries:
		return textPredicate("s.name", rule.Op, val)
	case domain.SmartFieldPublisher:
		return textPredicate("s.publisher", rule.Op, val)
	case domain.SmartFieldFormat:
		return textPredicate("b.file_format", rule.Op, val)
	case domain.SmartFieldAgeRating:
		return textPredicate("b.age_rating", rule.Op, val)

	case domain.SmartFieldReadStatus:
		pred, err := readStatusPredicate(val)
		if err != nil {
			return "", nil, err
		}
		return maybeNot(negate, pred), []any{userID}, nil
	}
	return "", nil, fmt.Errorf("%w: unknown rule field %q", domain.ErrValidation, rule.Field)
}

// textPredicate builds an is / isNot / contains comparison on a text column (case-insensitive).
func textPredicate(col, op, val string) (string, []any, error) {
	if val == "" {
		return "", nil, fmt.Errorf("%w: rule needs a value", domain.ErrValidation)
	}
	switch op {
	case domain.SmartOpIs:
		return col + " = ? COLLATE NOCASE", []any{val}, nil
	case domain.SmartOpIsNot:
		return "(" + col + " IS NULL OR " + col + " <> ? COLLATE NOCASE)", []any{val}, nil
	case domain.SmartOpContains:
		return col + " LIKE ? COLLATE NOCASE", []any{"%" + escapeLike(val) + "%"}, nil
	}
	return "", nil, fmt.Errorf("%w: unsupported operator %q", domain.ErrValidation, op)
}

// readStatusPredicate builds an EXISTS clause (bound to one user-id arg) for a read state.
func readStatusPredicate(status string) (string, error) {
	switch status {
	case string(domain.StatusRead):
		return `EXISTS (SELECT 1 FROM read_progress p WHERE p.book_id = b.id AND p.user_id = ? AND p.status = 'read')`, nil
	case string(domain.StatusInProgress):
		return `EXISTS (SELECT 1 FROM read_progress p WHERE p.book_id = b.id AND p.user_id = ? AND p.status = 'in_progress')`, nil
	case string(domain.StatusUnread):
		return `NOT EXISTS (SELECT 1 FROM read_progress p WHERE p.book_id = b.id AND p.user_id = ? AND p.status IN ('read','in_progress'))`, nil
	}
	return "", fmt.Errorf("%w: unknown read status %q", domain.ErrValidation, status)
}

func maybeNot(negate bool, pred string) string {
	if negate {
		return "NOT " + pred
	}
	return pred
}

// escapeLike neutralizes LIKE wildcards in user input (no ESCAPE clause needed for the
// default case since we only insert literal %).
func escapeLike(s string) string {
	return strings.NewReplacer("%", "", "_", "").Replace(s)
}

func scanSmartList(row rowScanner) (domain.SmartList, error) {
	var (
		l     domain.SmartList
		owner sql.NullString
		rules string
	)
	if err := row.Scan(&l.ID, &owner, &l.Name, &rules, &l.CreatedAt, &l.UpdatedAt); err != nil {
		return domain.SmartList{}, err
	}
	l.OwnerID = str(owner)
	if rules != "" {
		if err := json.Unmarshal([]byte(rules), &l.Rules); err != nil {
			return domain.SmartList{}, err
		}
	}
	return l, nil
}
