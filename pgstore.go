package iidy

import (
	"bytes"
	"strconv"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

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

func (p *PgStore) Add(listName string, itemID string) error {
	_, err := p.pool.Exec(`insert into lists
		(list, item)
		values ($1, $2)`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Get(listName string, itemID string) (uint, bool, error) {
	var attempts uint
	err := p.pool.QueryRow(`
		select attempts
		  from lists
		 where list = $1
		   and item = $2`, listName, itemID).Scan(&attempts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	return attempts, true, nil
}

func (p *PgStore) Del(listName string, itemID string) error {
	_, err := p.pool.Exec(`delete from lists
		where list = $1
		  and item = $2`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Inc(listName string, itemID string) error {
	_, err := p.pool.Exec(`update lists
		  set attempts = attempts + 1
		where list = $1
		  and item = $2`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) BulkAdd(listName string, itemIDs []string) error {
	if itemIDs == nil || len(itemIDs) == 0 {
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
	lastIndex := len(itemIDs) - 1
	for i, itemID := range itemIDs {
		buffer.WriteString("($")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		buffer.WriteString(", ")
		buffer.WriteString("$")
		args.Append(listName)
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		buffer.WriteString(")")
		if i < lastIndex {
			buffer.WriteString(",\n")
		}
		args.Append(itemID)
	}
	sql := buffer.String()
	_, err := p.pool.Exec(sql, args...)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) BulkGet(listName string, startID string, count int) ([]ListItem, error) {
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
		args.Append(listName)
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
		args.Append(listName)
		args.Append(startID)
		args.Append(count)
	}
	rows, err := p.pool.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// May as well grab as much mem as we need at the outset.
	items := make([]ListItem, 0, count)
	var item string
	var attempts uint
	for rows.Next() {
		err = rows.Scan(&item, &attempts)
		if err != nil {
			return nil, err
		}
		items = append(items, ListItem{Item: item, Attempts: attempts})
	}
	if rows.Err() != nil {
		return nil, err
	}
	return items, nil
}

func (p *PgStore) BulkDel(listName string, itemIDs []string) (int64, error) {
	if itemIDs == nil || len(itemIDs) == 0 {
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
	args.Append(listName)
	lastIndex := len(itemIDs) - 1
	for i, itemID := range itemIDs {
		buffer.WriteString("$")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		if i < lastIndex {
			buffer.WriteString(", ")
		}
		args.Append(itemID)
	}
	buffer.WriteString("]))")
	sql := buffer.String()
	commandTag, err := p.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}

func (p *PgStore) BulkInc(listName string, itemIDs []string) (int64, error) {
	if itemIDs == nil || len(itemIDs) == 0 {
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
	args.Append(listName)
	lastIndex := len(itemIDs) - 1
	for i, itemID := range itemIDs {
		buffer.WriteString("$")
		argNum++
		buffer.WriteString(strconv.Itoa(argNum))
		if i < lastIndex {
			buffer.WriteString(", ")
		}
		args.Append(itemID)
	}
	buffer.WriteString("]))")
	sql := buffer.String()
	commandTag, err := p.pool.Exec(sql, args...)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), nil
}
