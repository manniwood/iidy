[![](https://godoc.org/github.com/manniwood/iidy?status.svg)](https://godoc.org/github.com/manniwood/iidy)
[![Build Status](https://travis-ci.com/manniwood/iidy.svg)](https://travis-ci.com/manniwood/iidy)

# IIDY - Is It Done Yet?

IIDY is a simple yet scalable task list with a REST interface and a PostgreSQL
backend. It is still a little bit proof-of-concept, but it works pretty well.

To set up IIDY, all that should be required in terms of fetching packages is

    git clone git@github.com:manniwood/iidy.git

With that done, you need to have PostgreSQL installed an running at localhost:5432.
[This](https://www.manniwood.com/2017_02_27/postgresql_96_compile_install_howto.html)
is one way to accomplish that.

With PostgreSQL up and running, set up the iidy PostgreSQL user and database:

    cd $WHEREVER_YOU_CHECKED_OUT_IIDY/iidy/pg_setup
    psql -X -U postgres -d postgres -f setup.sql

And now, run IIDY:

    cd $WHEREVER_YOU_CHECKED_OUT_IIDY/iidy/cmd/iidy
    go build
    ./iidy

Now, you can play with IIDY through any HTTP client. Here are some examples
using curl:

    $ curl localhost:8080/lists/downloads/a.txt
    Not found.

    $ curl -X PUT localhost:8080/lists/downloads/a.txt

    $ curl localhost:8080/lists/downloads/a.txt
    0

    $ curl -X INCREMENT localhost:8080/lists/downloads/a.txt
    INCREMENTED 1

    $ curl localhost:8080/lists/downloads/a.txt
    1

    $ curl -X DELETE localhost:8080/lists/downloads/a.txt
    DELETED 1

    $ curl localhost:8080/lists/downloads/a.txt
    Not found.

    $ curl -X BULKPUT --data-binary @- localhost:8080/lists/downloads
    b.txt
    c.txt
    d.txt
    e.txt
    f.txt
    g.txt
    h.txt
    i.txt
    ^D
    ADDED 8

    $ curl -X BULKGET -H "X-IIDY-Count: 2" localhost:8080/lists/downloads
    b.txt 0
    c.txt 0

    $ curl -X BULKGET -H "X-IIDY-Count: 2" -H "X-IIDY-After-Item: c.txt" localhost:8080/lists/downloads
    d.txt 0
    e.txt 0

    $ curl -X BULKGET -H "X-IIDY-Count: 4" -H "X-IIDY-After-Item: e.txt" localhost:8080/lists/downloads
    f.txt 0
    g.txt 0
    h.txt 0
    i.txt 0

    $ curl -X BULKINCREMENT --data-binary @- localhost:8080/lists/downloads
    b.txt
    c.txt
    d.txt
    e.txt
    ^D
    INCREMENTED 4

    $ curl -X BULKGET -H "X-IIDY-Count: 100" localhost:8080/lists/downloads
    b.txt 1
    c.txt 1
    d.txt 1
    e.txt 1
    f.txt 0
    g.txt 0
    h.txt 0
    i.txt 0

    $ curl -X BULKDELETE --data-binary @- localhost:8080/lists/downloads
    d.txt
    e.txt
    f.txt
    g.txt
    ^D
    DELETED 4

    $ curl -X BULKGET -H "X-IIDY-Count: 100" localhost:8080/lists/downloads
    b.txt 1
    c.txt 1
    h.txt 0
    i.txt 0

