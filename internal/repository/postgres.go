package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repo struct {
	db *sql.DB
}

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func Migrate(db *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(ctx context.Context, s Subscription) (Subscription, error) {
	start, err := parseMonth(s.StartDate)
	if err != nil {
		return Subscription{}, err
	}

	var end sql.NullTime
	if s.EndDate != nil && *s.EndDate != "" {
		t, err := parseMonth(*s.EndDate)
		if err != nil {
			return Subscription{}, err
		}
		end = sql.NullTime{Time: t, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, s.ServiceName, s.Price, s.UserID, start, end)

	if err := row.Scan(&s.ID); err != nil {
		return Subscription{}, err
	}
	return s, nil
}

func (r *Repo) Get(ctx context.Context, id int) (Subscription, error) {
	var s Subscription
	var userID uuid.UUID
	var start, end sql.NullTime

	err := r.db.QueryRowContext(ctx, `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions WHERE id = $1
	`, id).Scan(&s.ID, &s.ServiceName, &s.Price, &userID, &start, &end)
	if err != nil {
		return Subscription{}, err
	}

	s.UserID = userID
	s.StartDate = formatMonth(start.Time)
	if end.Valid {
		v := formatMonth(end.Time)
		s.EndDate = &v
	}
	return s, nil
}

func (r *Repo) List(ctx context.Context, f Filter) ([]Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions WHERE 1=1
	`
	args := []any{}
	n := 1

	if f.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", n)
		args = append(args, *f.UserID)
		n++
	}
	if f.ServiceName != nil {
		query += fmt.Sprintf(" AND service_name = $%d", n)
		args = append(args, *f.ServiceName)
		n++
	}
	query += " ORDER BY id"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		s, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repo) Update(ctx context.Context, id int, s Subscription) error {
	start, err := parseMonth(s.StartDate)
	if err != nil {
		return err
	}

	var end sql.NullTime
	if s.EndDate != nil && *s.EndDate != "" {
		t, err := parseMonth(*s.EndDate)
		if err != nil {
			return err
		}
		end = sql.NullTime{Time: t, Valid: true}
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE subscriptions
		SET service_name = $1, price = $2, user_id = $3, start_date = $4, end_date = $5
		WHERE id = $6
	`, s.ServiceName, s.Price, s.UserID, start, end, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repo) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Total считает сумму: price * кол-во месяцев пересечения подписки с периодом
func (r *Repo) Total(ctx context.Context, f PeriodFilter) (int, error) {
	list, err := r.List(ctx, Filter{UserID: f.UserID, ServiceName: f.ServiceName})
	if err != nil {
		return 0, err
	}

	total := 0
	for _, s := range list {
		start, _ := parseMonth(s.StartDate)
		var end time.Time
		if s.EndDate != nil && *s.EndDate != "" {
			end, _ = parseMonth(*s.EndDate)
		} else {
			end = f.To
		}

		months := overlapMonths(start, end, f.From, f.To)
		if months > 0 {
			total += s.Price * months
		}
	}
	return total, nil
}

func scanSub(rows *sql.Rows) (Subscription, error) {
	var s Subscription
	var userID uuid.UUID
	var start time.Time
	var end sql.NullTime

	if err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &userID, &start, &end); err != nil {
		return Subscription{}, err
	}
	s.UserID = userID
	s.StartDate = formatMonth(start)
	if end.Valid {
		v := formatMonth(end.Time)
		s.EndDate = &v
	}
	return s, nil
}

func ParseMonth(v string) (time.Time, error) {
	return parseMonth(v)
}

func parseMonth(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	t, err := time.Parse("01-2006", v)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format, use MM-YYYY")
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), nil
}

func formatMonth(t time.Time) string {
	return t.Format("01-2006")
}

func overlapMonths(aStart, aEnd, bStart, bEnd time.Time) int {
	start := maxTime(aStart, bStart)
	end := minTime(aEnd, bEnd)
	if end.Before(start) {
		return 0
	}
	months := 0
	cur := start
	for !cur.After(end) {
		months++
		cur = cur.AddDate(0, 1, 0)
	}
	return months
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
