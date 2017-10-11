package iidy

import (
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type PgStore struct {
	// need a pg connection pool here
	pool *pgx.ConnPool
}

func NewPgStore() (Store, error) {
	// TODO: make this configurable
	conf := pgx.ConnConfig{Host: "localhost", Database: "iidy", User: "iidy"}
	pconf := pgx.ConnPoolConfig{conf, MaxConnections: 5}
	pool, err := pgx.NewConnPool(pconf)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create PgStore")
	}
	p := PgStore{pool: pool}
	return &p, nil
}

func (p *PgStore) Add(listName string, itemID string) (err error) {
	commandTag, err := p.pool.Exec(`insert into lists
		(list, item)
		values ($1, $2)`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Get(listName string, itemID string) (uint, bool, error) {
	// XXX START HERE; how do we know if this returned nothing? We need to
	// know, because 0 is a legit value for attempts. Ah, we need to
	// do .Scan() later, because the return type *Row will error with ErrNoRows if no rows are returned.
	// or maybe that just gets passed through so we can keep them connected
	// but we should check for that exact error below.
	var attempts uint
	err := p.pool.QueryRow(`
		select attempts
		  from lists
		 where list = $1
		   and item = $2`, listName, itemID).Scan(&attempts)
	if err != nil {
		return 0, false, err
	}
	return 0, false, nil
}

func (p *PgStore) Del(listName string, itemID string) (err error) {
	commandTag, err := p.pool.Exec(`delete from lists
		where list = $1
		  and item = $2`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}

func (p *PgStore) Inc(listName string, itemID string) (err error) {
	commandTag, err := p.pool.Exec(`update lists
		  set attempts = attempts + 1
		where list = $1
		  and item = $2`, listName, itemID)
	if err != nil {
		return err
	}
	return nil
}
