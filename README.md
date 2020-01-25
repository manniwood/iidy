[![](https://godoc.org/github.com/manniwood/iidy?status.svg)](https://godoc.org/github.com/manniwood/iidy)
[![Build Status](https://travis-ci.com/manniwood/iidy.svg)](https://travis-ci.com/manniwood/iidy)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# IIDY - Is It Done Yet?

IIDY is a little project I set up to play with Go. It's meant to explore ideas
more than it is meant to be a production-ready product. One restriction I set
for myself was to see how much I could accomplish using just the Go standard
library.

IIDY is a simple yet scalable task list (attempt list) with a PostgreSQL
backend. It also provides a REST API.

The basic problem IIDY wants to solve is to be an attempt list for millions
or billions of items. An example use case is trying to download a few million
files, where not every attempt will be initially successful. One would want to
track how many attempts were made per file, in addition to removing files whose
downloads were successful.

## Setup

To set up IIDY, all that should be required in terms of fetching packages is

```
git clone git@github.com:manniwood/iidy.git
```

With that done, you need to have PostgreSQL installed an running at localhost:5432.
[This](https://www.manniwood.com/2017_02_27/postgresql_96_compile_install_howto.html)
is one way to accomplish that.

With PostgreSQL up and running, set up the iidy PostgreSQL user and database:

```
cd $WHEREVER_YOU_CHECKED_OUT_IIDY/iidy/pg_setup
psql -X -U postgres -d postgres -f setup.sql
```

And now, run IIDY:

```
cd $WHEREVER_YOU_CHECKED_OUT_IIDY/iidy/cmd/iidy
go build
./iidy
```

Now, you can play with IIDY through any HTTP client. Here are some examples
using curl:

```
$ curl localhost:8080/iidy/v1/lists/downloads/a.txt
Not found.

$ curl -X POST localhost:8080/iidy/v1/lists/downloads/a.txt
ADDED 1

$ curl localhost:8080/iidy/v1/lists/downloads/a.txt
0

$ curl -X POST localhost:8080/iidy/v1/lists/downloads/a.txt?action=increment
INCREMENTED 1

$ curl localhost:8080/iidy/v1/lists/downloads/a.txt
1

$ curl -X DELETE localhost:8080/iidy/v1/lists/downloads/a.txt
DELETED 1

$ curl localhost:8080/iidy/v1/lists/downloads/a.txt
Not found.

$ curl -X POST localhost:8080/iidy/v1/bulk/lists/downloads -d '
b.txt
c.txt
d.txt
e.txt
f.txt
g.txt
h.txt
i.txt
'
ADDED 8

$ curl -H "X-IIDY-Count: 2" localhost:8080/iidy/v1/bulk/lists/downloads
b.txt 0
c.txt 0

$ curl -H "X-IIDY-Count: 2" -H "X-IIDY-After-Item: c.txt" localhost:8080/iidy/v1/bulk/lists/downloads
d.txt 0
e.txt 0

$ curl -H "X-IIDY-Count: 4" -H "X-IIDY-After-Item: e.txt" localhost:8080/iidy/v1/bulk/lists/downloads
f.txt 0
g.txt 0
h.txt 0
i.txt 0

$ curl localhost:8080/iidy/v1/bulk/lists/downloads?action=increment -d '
b.txt
c.txt
d.txt
e.txt
'
INCREMENTED 4

$ curl -H "X-IIDY-Count: 100" localhost:8080/iidy/v1/bulk/lists/downloads
b.txt 1
c.txt 1
d.txt 1
e.txt 1
f.txt 0
g.txt 0
h.txt 0
i.txt 0

$ curl -X DELETE localhost:8080/iidy/v1/bulk/lists/downloads -d '
d.txt
e.txt
f.txt
g.txt
'
DELETED 4

$ curl -H "X-IIDY-Count: 100" localhost:8080/iidy/v1/bulk/lists/downloads
b.txt 1
c.txt 1
h.txt 0
i.txt 0
```


