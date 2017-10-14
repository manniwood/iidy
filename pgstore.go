package iidy

import (
	"bytes"
	"strconv"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type ListEntry struct {
	Item     string
	Attempts int
}

type PgStore struct {
	// need a pg connection pool here
	pool *pgx.ConnPool
}

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

func (p *PgStore) Nuke() error {
	_, err := p.pool.Exec(`truncate table lists`)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Add(list string, item string) error {
	_, err := p.pool.Exec(`insert into lists
		(list, item)
		values ($1, $2)`, list, item)
	if err != nil {
		return err
	}
	return nil
}

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

func (p *PgStore) Del(list string, item string) error {
	_, err := p.pool.Exec(`delete from lists
		where list = $1
		  and item = $2`, list, item)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Inc(list string, item string) error {
	_, err := p.pool.Exec(`update lists
		  set attempts = attempts + 1
		where list = $1
		  and item = $2`, list, item)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) BulkAdd(list string, items []string) error {
	if items == nil || len(items) == 0 {
		return nil
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
	_, err := p.pool.Exec(sql, args...)
	if err != nil {
		return err
	}
	return nil
}

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
