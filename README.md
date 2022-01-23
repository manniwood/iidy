[![Go Reference](https://pkg.go.dev/badge/github.com/manniwood/iidy.svg)](https://pkg.go.dev/github.com/manniwood/iidy)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

# IIDY - Is It Done Yet?

## Status: For Play Purposes Only

This code is not intended for production use. It is a fun way to explore
some ideas with Go and PostgreSQL. It is very permissively licenced, so
feel free to beg/borrow/steal anything that you like from here.

## Summary

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
[This](https://www.manniwood.com/2021_10_29/postgresql_14_compile_install_howto.html)
is one way to accomplish that.

With PostgreSQL up and running, set up the iidy PostgreSQL user and database:

And now, run IIDY:

```
cd $WHEREVER_YOU_CHECKED_OUT_IIDY/iidy/migrations
go install github.com/jackc/tern@latest
tern migrate
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

$ curl -X POST localhost:8080/iidy/v1/batch/lists/downloads -d '
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

$ curl localhost:8080/iidy/v1/batch/lists/downloads?count=2
b.txt 0
c.txt 0

$ curl "localhost:8080/iidy/v1/batch/lists/downloads?count=2&after_id=c.txt"
d.txt 0
e.txt 0

$ curl "localhost:8080/iidy/v1/batch/lists/downloads?count=4&after_id=e.txt"
f.txt 0
g.txt 0
h.txt 0
i.txt 0

$ curl localhost:8080/iidy/v1/batch/lists/downloads?action=increment -d '
b.txt
c.txt
d.txt
e.txt
'
INCREMENTED 4

$ curl localhost:8080/iidy/v1/batch/lists/downloads?count=100
b.txt 1
c.txt 1
d.txt 1
e.txt 1
f.txt 0
g.txt 0
h.txt 0
i.txt 0

$ curl -X DELETE localhost:8080/iidy/v1/batch/lists/downloads -d '
d.txt
e.txt
f.txt
g.txt
'
DELETED 4

$ curl localhost:8080/iidy/v1/batch/lists/downloads?count=100
b.txt 1
c.txt 1
h.txt 0
i.txt 0
```


Here are the same examples using JSON:

```
$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"error":"Not found."}

$ curl -X POST -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"added":1}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"item":"a.txt","attempts":0}

$ curl -X POST -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt?action=increment
{"incremented":1}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"item":"a.txt","attempts":1}

$ curl -X DELETE -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"deleted":1}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/lists/downloads/a.txt
{"error":"Not found."}

$ curl -X POST -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads -d '
{"items":[
"b.txt",
"c.txt",
"d.txt",
"e.txt",
"f.txt",
"g.txt",
"h.txt",
"i.txt"]}
'
{"added":8}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads?count=2
{"listentries":[
{"item":"b.txt","attempts":0},
{"item":"c.txt","attempts":0}]}

$ curl -H "Content-type: application/json" "localhost:8080/iidy/v1/batch/lists/downloads?count=2&after_id=c.txt"
{"listentries":[
{"item":"d.txt","attempts":0},
{"item":"e.txt","attempts":0}]}

$ curl -H "Content-type: application/json" "localhost:8080/iidy/v1/batch/lists/downloads?count=4&after_id=e.txt"
{"listentries":[
{"item":"f.txt","attempts":0},
{"item":"g.txt","attempts":0},
{"item":"h.txt","attempts":0},
{"item":"i.txt","attempts":0}]}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads?action=increment -d '
{"items":[
"b.txt",
"c.txt",
"d.txt",
"e.txt"]}
'
{"incremented":4}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads?count=100
{"listentries":[
{"item":"b.txt","attempts":1},
{"item":"c.txt","attempts":1},
{"item":"d.txt","attempts":1},
{"item":"e.txt","attempts":1},
{"item":"f.txt","attempts":0},
{"item":"g.txt","attempts":0},
{"item":"h.txt","attempts":0},
{"item":"i.txt","attempts":0}]}

$ curl -X DELETE -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads -d '
{"items":[
"d.txt",
"e.txt",
"f.txt",
"g.txt"]}
'
{"deleted":4}

$ curl -H "Content-type: application/json" localhost:8080/iidy/v1/batch/lists/downloads?count=100
{"listentries":[
{"item":"b.txt","attempts":1},
{"item":"c.txt","attempts":1},
{"item":"h.txt","attempts":0},
{"item":"i.txt","attempts":0}]}
```



