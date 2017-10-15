package iidy

import (
	"bytes"
	"strconv"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

// ListEntry is a list item and the number of times
// an attempt has been made to complete it.
type ListEntry struct {
	Item     string
	Attempts int
}

// PgStore is the backend store where lists and their
// items are kept.
type PgStore struct {
	pool *pgx.ConnPool
}

// NewPgStore returns a pointer to a new PgStore.
// It's best to treat an instance of PgStore like
// a singleton, and have only one per process.
func NewPgStore() (*PgStore, error) {
	// TODO: make this configurable
	conf := pgx.ConnConfig{Host: "localhost", Database: "iidy", User: "iidy"}
	pconf := pgx.ConnPoolConfig{ConnConfig: conf, MaxConnections: 5}
	pool, err := pgx.NewConnPool(pconf)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create PgStore")
	}
	p := PgStore{pool: pool}
	return &p, nil
}

// Nuke will destroy every list in the data store.
// Use with caution.
func (p *PgStore) Nuke() error {
	_, err := p.pool.Exec(`truncate table lists`)
	if err != nil {
		return err
	}
	return nil
}

// Add adds an item to a list. If the list does not already
// exist, it will be created.
func (p *PgStore) Add(list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(`
		insert into lists
		(list, item)
		values ($1, $2)`, list, item)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// Get returns the number of attempts that were made to
// complete an item in a list. When a list or list item
// is missing, the number of attempts will be returned
// as 0, but the second return argument (commonly assiged
// to "ok") will be false.
func (p *PgStore) Get(list string, item string) (int, bool, error) {
	var attempts int
	err := p.pool.QueryRow(`
		select attempts
		  from lists
		 where list = $1
		   and item = $2`, list, item).Scan(&attempts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return attempts, true, nil
}

// Del deletes an item from a list. The first return value
// is the number of items found and deleted (1 or 0).
func (p *PgStore) Del(list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(`
		delete from lists
		 where list = $1
		   and item = $2`, list, item)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// Inc increments the number of attempts to complete
// an item from a list. The first return value
// is the number of items found and incremented (1 or 0).
func (p *PgStore) Inc(list string, item string) (int64, error) {
	commandTag, err := p.pool.Exec(`
		update lists
		   set attempts = attempts + 1
		 where list = $1
		   and item = $2`, list, item)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// BulkAdd adds a slice of items (strings) to the specified
// list, and sets their completion attempt counts to 0.
// The first return value is the number of items successfully
// inserted, generally len(items) or 0.
func (p *PgStore) BulkAdd(list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	// The query we need to build looks like this:
	// insert into lists
	// (list, item)
	// values
	// ($1, $2),
	// ($3, $4),
	// ...
	// ($11, $12) <-- no trailing comma
	var buffer bytes.Buffer
	buffer.WriteString("insert into lists (list, item) values \n")
	argNum := 0
	args := make(pgx.QueryArgs, 0)
	lastIndex := len(items) - 1
	for i, item := range items {
		buffer.WriteString("($")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		buffer.WriteString(", ")
		buffer.WriteString("$")
		args.Append(list)
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		buffer.WriteString(")")
		if i < lastIndex {
			buffer.WriteString(",\n")
		}
		args.Append(item)
	}
	sql := buffer.String()
	commandTag, err := p.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// BulkGet gets a slice of ListEntries from the specified
// list (alphabetically sorted), starting after the startID,
// or from the beginning of the list, if startID is an empty string.
// If there is nothing to be found, an empty slice is returned.
//
// The general pattern being followed here is explained very well at
// http://use-the-index-luke.com/sql/partial-results/fetch-next-page
func (p *PgStore) BulkGet(list string, startID string, count int) ([]ListEntry, error) {
	if count == 0 {
		return []ListEntry{}, nil
	}
	var sql string
	args := make(pgx.QueryArgs, 0)
	if startID == "" {
		sql = `
		  select item,
				 attempts
			from lists
		   where list = $1
		order by list,
				 item
		   limit $2`
		args.Append(list)
		args.Append(count)
	} else {
		sql = `
		  select item,
				 attempts
		    from lists
		   where list = $1
			 and item > $2
		order by list,
				 item
		   limit $3`
		args.Append(list)
		args.Append(startID)
		args.Append(count)
	}
	rows, err := p.pool.Query(sql, args...)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		items = append(items, ListEntry{Item: item, Attempts: attempts})
	}
	if rows.Err() != nil {
		return nil, err
	}
	return items, nil
}

// BulkDel deletes a slice of items (strings) from the specified
// list. The first return value is the number of items successfully
// deleted, generally len(items) or 0.
func (p *PgStore) BulkDel(list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	// The query we need to build looks like this:
	// delete from lists
	//       where list = $1
	//         and item in (select unnest(array[$2, $3, ... $12]))"
	var buffer bytes.Buffer
	buffer.WriteString(`
		delete from lists
		      where list = $1
		        and item in (select unnest(array[`)
	argNum := 1
	args := make(pgx.QueryArgs, 0)
	args.Append(list)
	lastIndex := len(items) - 1
	for i, item := range items {
		buffer.WriteString("$")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		if i < lastIndex {
			buffer.WriteString(", ")
		}
		args.Append(item)
	}
	buffer.WriteString("]))")
	sql := buffer.String()
	commandTag, err := p.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

// BulkInc increments the attempts count for each item in the items
// slice for the specified list.  The first return value is the number
// of items successfully incremented, generally len(items) or 0.
func (p *PgStore) BulkInc(list string, items []string) (int64, error) {
	if items == nil || len(items) == 0 {
		return 0, nil
	}
	// The query we need to build looks like this:
	// update lists
	//    set attempts = attempts + 1
	//       where list = $1
	//         and item in (select unnest(array[$2, $3, ... $12]))"
	var buffer bytes.Buffer
	buffer.WriteString(`
		update lists
		   set attempts = attempts + 1
	     where list = $1
	       and item in (select unnest(array[`)
	argNum := 1
	args := make(pgx.QueryArgs, 0)
	args.Append(list)
	lastIndex := len(items) - 1
	for i, item := range items {
		buffer.WriteString("$")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		if i < lastIndex {
			buffer.WriteString(", ")
		}
		args.Append(item)
	}
	buffer.WriteString("]))")
	sql := buffer.String()
	commandTag, err := p.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}
