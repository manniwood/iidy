package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NOTE on error handling: we follow the advice at https://blog.golang.org/go1.13-errors:
// The pgx errors we will be dealing with are internal details.
// To avoid exposing them to the caller, we repackage them as new
// errors with the same text. We use the %v formatting verb, since
// %w would permit the caller to unwrap the original pgx errors.
// We don't want to support pgx errors as part of our API.

// DefaultConnectionURL is the default connection URL
// to the PostgreSQL database, including connection pool
// config and application_name config.
const DefaultConnectionURL string = "postgresql://postgres:postgres@localhost:5432/postgres?pool_max_conns=5&application_name=iidy"
const TernDefaultMigrationTable string = "public.schema_version"

// itemCopier implements pgx.CopyFromSource. It can be used to copy a
// slice of Items into the named List.
type itemCopier struct {
	List  string
	Items []string
	Len   int
	I     int
}

// newItemCopier constructs a new itemCopier
func newItemCopier(list string, items []string) *itemCopier {
	return &itemCopier{
		List:  list,
		Items: items,
		Len:   len(items),
		I:     0,
	}
}

// Next tells pgx if there is another row of input left to
// copy into the destination table.
func (cp *itemCopier) Next() bool {
	return cp.I < cp.Len
}

// Values is called by a pgx copy command when it is ready
// for the next row of input.
func (cp *itemCopier) Values() ([]interface{}, error) {
	row := []interface{}{cp.List, cp.Items[cp.I]}
	cp.I++
	return row, nil
}

// Err can be called if there were any errors encountered
// while copying.
func (cp *itemCopier) Err() error {
	return nil
}

// ListEntry is a list item and the number of times an attempt has been
// made to complete it.
type ListEntry struct {
	Item     string `json:"item"`
	Attempts int    `json:"attempts"`
}

// Store describes list storage methods, in case we want to
// have a different implementation than the pg implementation.
type Store interface {
	InsertOne(ctx context.Context, list string, item string) (int64, error)
	GetOne(ctx context.Context, list string, item string) (int, bool, error)
	DeleteOne(ctx context.Context, list string, item string) (int64, error)
	IncrementOne(ctx context.Context, list string, item string) (int64, error)
	InsertBatch(ctx context.Context, list string, items []string) (int64, error)
	GetBatch(ctx context.Context, list string, startID string, count int) ([]ListEntry, error)
	DeleteBatch(ctx context.Context, list string, items []string) (int64, error)
	IncrementBatch(ctx context.Context, list string, items []string) (int64, error)
}

// PgStore is the backend store where lists and list items are kept.
type PgStore struct {
	connectionURL string
	pool          *pgxpool.Pool
}

// NewPgStore returns a pointer to a new PgStore. It's best to treat an
// instance of PgStore like a singleton, and have only one per process.
// connectionURL is a connection string is formatted like so,
//
//     postgresql://[user[:password]@][netloc][:port][,...][/dbname][?param1=value1&...]
//
// according to https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING.
//
// If connectionURL is the empty string, DefaultConnectionURL will be used.
func NewPgStore(connectionURL string) (*PgStore, error) {
	if connectionURL == "" {
		connectionURL = DefaultConnectionURL
	}
	pool, err := pgxpool.Connect(context.Background(), connectionURL)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	p := PgStore{
		connectionURL: connectionURL,
		pool:          pool,
	}
	return &p, nil
}

// String gives us a string representation of the config for the data store.
// This is handy for debugging, or just for printing the connection info
// at program startup.
func (p *PgStore) String() string {
	conf, err := pgxpool.ParseConfig(p.connectionURL)
	if err != nil {
		return fmt.Sprintf("Could not parse connection URL: %v", err)
	}
	c := conf.ConnConfig.Config
	return fmt.Sprintf(`
Host: %s
Port: %d
DB:   %s
User: %s
`,
		c.Host,
		c.Port,
		c.Database,
		c.User,
	)
}

// Nuke destroys every list in the data store. Mostly used for testing.
// Use with caution.
func (p *PgStore) Nuke(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `truncate table iidy.lists`)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	return nil
}

// InsertOne adds an item to a list. If the list does not already exist,
// it will be created.
func (p *PgStore) InsertOne(ctx context.Context, list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(ctx, `
		insert into iidy.lists
		(list, item)
		values ($1, $2)`, list, item)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}

// GetOne returns the number of attempts that were made to complete an item
// in a list. When a list or list item is missing, the number of attempts
// will be returned as 0, but the second return argument (commonly assiged
// to "ok") will be false.
func (p *PgStore) GetOne(ctx context.Context, list string, item string) (int, bool, error) {
	var attempts int
	err := p.pool.QueryRow(ctx, `
		select attempts
		  from iidy.lists
		 where list = $1
		   and item = $2`, list, item).Scan(&attempts)
	if err != nil {
		// using `errors.Is()` is more robust than `if err == pgx.ErrNoRows`
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("%v", err)
	}
	return attempts, true, nil
}

// DeleteOne deletes an item from a list. The first return value is the number of
// items that were successfully deleted (1 or 0).
func (p *PgStore) DeleteOne(ctx context.Context, list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(ctx, `
		delete from iidy.lists
		 where list = $1
		   and item = $2`, list, item)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}

// IncrementOne increments the number of attempts to complete an item from a list.
// The first return value is the number of items found and incremented
// (1 or 0).
func (p *PgStore) IncrementOne(ctx context.Context, list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(ctx, `
		update iidy.lists
		   set attempts = attempts + 1
		 where list = $1
		   and item = $2`, list, item)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}

// InsertBatch adds a slice of items (strings) to the specified list, and sets
// their completion attempt counts to 0. The first return value is the
// number of items successfully inserted, generally len(items) or 0.
func (p *PgStore) InsertBatch(ctx context.Context, list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	copyCount, err := p.pool.CopyFrom(
		ctx,
		pgx.Identifier{"iidy", "lists"},
		[]string{"list", "item"},
		newItemCopier(list, items))
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return copyCount, nil
}

// GetBatch gets a slice of ListEntries from the specified list
// (alphabetically sorted), starting after the startID, or from the beginning
// of the list, if startID is an empty string. If there is nothing to be found,
// an empty slice is returned.
//
// The general pattern being followed here is explained very well at
// http://use-the-index-luke.com/sql/partial-results/fetch-next-page
func (p *PgStore) GetBatch(ctx context.Context, list string, startID string, count int) ([]ListEntry, error) {
	if count == 0 {
		return []ListEntry{}, nil
	}
	var rows pgx.Rows
	var err error
	if startID == "" {
		sql := `
      select item,
             attempts
        from iidy.lists
       where list = $1
    order by list,
             item
       limit $2`
		rows, err = p.pool.Query(ctx, sql, list, count)
	} else {
		sql := `
      select item,
             attempts
        from iidy.lists
       where list = $1
         and item > $3
    order by list,
             item
       limit $2`
		rows, err = p.pool.Query(ctx, sql, list, count, startID)
	}
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}
	defer rows.Close()

	// Up front, may as well allocate as much memory
	// as we need for the entire list.
	items := make([]ListEntry, 0, count)
	var item string
	var attempts int
	for rows.Next() {
		err = rows.Scan(&item, &attempts)
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}
		items = append(items, ListEntry{Item: item, Attempts: attempts})
	}
	if rows.Err() != nil {
		return nil, err
	}
	return items, nil
}

// DeleteBatch deletes a slice of items (strings) from the specified list.
// The first return value is the number of items successfully deleted,
// generally len(items) or 0.
func (p *PgStore) DeleteBatch(ctx context.Context, list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	// pgx is smart enough to convert `items []string` into postgresql's text[],
	// which is very nice, because then we can use `items []string` as a single
	// parameter in the SQL query (`$2`) instead of needing a bunch of parameters
	// (`$2, $3, $4, ...`).
	// We could have done `and item = any($2)` but see
	// https://www.manniwood.com/2016_02_01/arrays_and_the_postgresql_query_planner.html
	// for why unnesting the array into a table makes the query planner happier.
	sql := `
		delete from iidy.lists
		      where list = $1
						and item in (select unnest($2::text[]))`
	commandTag, err := p.pool.Exec(ctx, sql, list, items)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}

// IncrementBatch increments the attempts count for each item in the items slice for
// the specified list.  The first return value is the number of items
// successfully incremented, generally len(items) or 0.
func (p *PgStore) IncrementBatch(ctx context.Context, list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	// pgx is smart enough to convert `items []string` into postgresql's text[],
	// which is very nice, because then we can use `items []string` as a single
	// parameter in the SQL query (`$2`) instead of needing a bunch of parameters
	// (`$2, $3, $4, ...`).
	// We could have done `and item = any($2)` but see
	// https://www.manniwood.com/2016_02_01/arrays_and_the_postgresql_query_planner.html
	// for why unnesting the array into a table makes the query planner happier.
	sql := `
		update iidy.lists
		   set attempts = attempts + 1
	     where list = $1
				and item in (select unnest($2::text[]))`
	commandTag, err := p.pool.Exec(ctx, sql, list, items)
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}
