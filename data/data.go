package data

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/manniwood/pgxtras"
)

// NOTE on error handling: we follow the advice at https://blog.golang.org/go1.13-errors:
// The pgx errors we will be dealing with are internal details.
// To avoid exposing them to the caller, we repackage them as new
// errors with the same text. We use the %v formatting verb, since
// %w would permit the caller to unwrap the original pgx errors.
// We don't want to support pgx errors as part of our API.

// DefaultPgPoolURL is the default connection URL
// to the PostgreSQL database, including connection pool
// config and application_name config.
const DefaultPgPoolURL string = "postgresql://postgres:postgres@localhost:5432/postgres?pool_max_conns=5"
const DefaultPgMigrationURL string = "postgresql://postgres:postgres@localhost:5432/postgres"
const TernMigrationTable string = "public.iidy_schema_version"

// At application startup, a pgx pool will get created and assigned
// to this var.
var PgxPool *pgxpool.Pool

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

func CreatePGXPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("problem parsing pool db URL %s: %w", dbURL, err)
	}

	// Identify app pool connections as "iidy" in pg_stat_activity
	connConfig.ConnConfig.RuntimeParams["application_name"] = "iidy"

	conn, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("problem making pool connection to db: %w", err)
	}
	return conn, nil
}

func CreatePGXConnForMigration(ctx context.Context, dbURL string) (*pgx.Conn, error) {
	connConfig, err := pgx.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("problem parsing migration db URL %s: %w", dbURL, err)
	}

	// Identify migration connection as "iidy_migration" in pg_stat_activity
	connConfig.RuntimeParams["application_name"] = "iidy_migration"

	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("problem making migration connection to db: %w", err)
	}
	return conn, nil
}

func MigrateDB(ctx context.Context, conn *pgx.Conn, migrationFilesDir embed.FS, migrationTableName string) error {
	if migrationTableName == "" {
		return fmt.Errorf("migration table name not set")
	}

	migrator, err := migrate.NewMigrator(ctx, conn, migrationTableName)
	if err != nil {
		return fmt.Errorf("failed to construct database migrator: %w", err)
	}

	err = migrator.LoadMigrations(migrationFilesDir)
	if err != nil {
		return fmt.Errorf("failed to load migration files: %w", err)
	}

	err = migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}
	return nil
}

// InsertOne adds an item to a list. If the list does not already exist,
// it will be created.
func InsertOne(ctx context.Context, e Execer, list string, item string) (int64, error) {
	commandTag, err := e.Exec(ctx, `
		insert into iidy.lists
		(list, item)
		values (@list, @item)`, pgx.NamedArgs{"list": list, "item": item})
	// NOTE: It is safe to dereference commandTag even when err is non-nil
	// because Exec returns a pgconn.CommandTag, NOT *pgconn.CommandTag
	return commandTag.RowsAffected(), err
}

// GetOne returns the number of attempts that were made to complete an item
// in a list. When a list or list item is missing, the number of attempts
// will be returned as 0, but the second return argument (commonly assiged
// to "ok") will be false.
func GetOne(ctx context.Context, q Querier, list string, item string) (int, bool, error) {
	var attempts int
	rowFetcher, err := q.Query(ctx, `
		select attempts
		  from iidy.lists
		 where list = @list 
		 and item = @item`, pgx.NamedArgs{"list": list, "item": item})
	if err != nil {
		return 0, false, err
	}
	attempts, ok, err := pgxtras.CollectOneRowOK(rowFetcher, pgx.RowTo[int])
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return attempts, true, nil
}

// DeleteOne deletes an item from a list. The first return value is the number of
// items that were successfully deleted (1 or 0).
func DeleteOne(ctx context.Context, e Execer, list string, item string) (int64, error) {
	commandTag, err := e.Exec(ctx, `
		delete from iidy.lists
		 where list = @list
		 and item = @item`, pgx.NamedArgs{"list": list, "item": item})
	return commandTag.RowsAffected(), err
}

// IncrementOne increments the number of attempts to complete an item from a list.
// The first return value is the number of items found and incremented
// (1 or 0).
func IncrementOne(ctx context.Context, e Execer, list string, item string) (int64, error) {
	commandTag, err := e.Exec(ctx, `
		update iidy.lists
		   set attempts = attempts + 1
		 where list = @list
		 and item = @item`, pgx.NamedArgs{"list": list, "item": item})
	if err != nil {
		return 0, fmt.Errorf("%v", err)
	}
	return commandTag.RowsAffected(), nil
}

// InsertBatch adds a slice of items (strings) to the specified list, and sets
// their completion attempt counts to 0. The first return value is the
// number of items successfully inserted, generally len(items) or 0.
func InsertBatch(ctx context.Context, cf CopyFromer, list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	return cf.CopyFrom(
		ctx,
		pgx.Identifier{"iidy", "lists"},
		[]string{"list", "item"},
		newItemCopier(list, items))
}

// GetBatch gets a slice of ListEntries from the specified list
// (alphabetically sorted), starting after the startID, or from the beginning
// of the list, if startID is an empty string. If there is nothing to be found,
// an empty slice is returned.
//
// The general pattern being followed here is explained very well at
// http://use-the-index-luke.com/sql/partial-results/fetch-next-page
func GetBatch(ctx context.Context, q Querier, list string, startID string, count int) ([]ListEntry, error) {
	if count == 0 {
		return []ListEntry{}, nil
	}
	var rowFetcher pgx.Rows
	var err error
	if startID == "" {
		sql := `
      select item,
             attempts
        from iidy.lists
       where list = @list
    order by list,
             item
       limit @count`
		rowFetcher, err = q.Query(ctx, sql, pgx.NamedArgs{"list": list, "count": count})
	} else {
		sql := `
      select item,
             attempts
        from iidy.lists
       where list = @list
         and item > @startID
    order by list,
             item
       limit @count`
		rowFetcher, err = q.Query(ctx, sql, pgx.NamedArgs{"list": list, "startID": startID, "count": count})
	}
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rowFetcher, pgxtras.RowToStructBySimpleName[ListEntry])
}

// DeleteBatch deletes a slice of items (strings) from the specified list.
// The first return value is the number of items successfully deleted,
// generally len(items) or 0.
func DeleteBatch(ctx context.Context, e Execer, list string, items []string) (int64, error) {
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
          where list = @list
            and item in (select unnest(@items::text[]))`
	commandTag, err := e.Exec(ctx, sql, pgx.NamedArgs{"list": list, "items": items})
	/*
			sql := `
		    delete from iidy.lists
		          where list = $1
		            and item in (select unnest($2::text[]))`
			commandTag, err := e.Exec(ctx, sql, list, items)
	*/
	return commandTag.RowsAffected(), err
}

// IncrementBatch increments the attempts count for each item in the items slice for
// the specified list.  The first return value is the number of items
// successfully incremented, generally len(items) or 0.
func IncrementBatch(ctx context.Context, e Execer, list string, items []string) (int64, error) {
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
     where list = @list
       and item in (select unnest(@items::text[]))`
	commandTag, err := e.Exec(ctx, sql, pgx.NamedArgs{"list": list, "items": items})
	return commandTag.RowsAffected(), err
}
