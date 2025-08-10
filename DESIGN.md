# Notes on the design of IIDY

IIDY is a pet project where I explore API/service ideas in Go.

## Purpose

Ever work on a project where there were millions or billions of items to
process, some of which would succeed, and some of which would fail?
And you needed a way to track the items to be processed, so that
items that failed the first attempt to be processed could be retried
until successful?

IIDY is an exploration of that problem.

Let's say you had to download a million files.

You could follow a workflow like this:

1) Add to a list the names of files to be downloaded.

2) Attempt to download the files, and 

a) when successful, delete the names of the successfully-downloaded
files from the list we made in step 1.

b) when failed, increment the number of attempts for each failed file
from the list we made in step 1.

3) Try failed items again later. Abandon items that failed 
a certain number of times, by deleting them from the list.

4) Eventually, the list of items to be processed ends up empty.

One could also follow more complex workflows, where a co-ordinator
might assign non-overlapping ranges of downloads to workers
to procdess in parallel.

That's the problem IIDY tries to solve.

Usually this problem would not stand alone in as its own project, but this
is for fun/education.

## Constraint

How much can I get done using just the Go standard library?
I definitely need a library to connect to PostgreSQL (the data store
I used for this project) but otherwise, everything else can
be done using Go's standard library.

## DB

One way to keep track of this in an RDBMS is through a simple
table where we track the name of the list alongside the name of
an item in that list, and the number of attempts made to process
that item.

A single table is all that is be required.

```
create table lists (
	list     text    not null,
	item     text    not null,
	attempts integer not null default 0,
	constraint list_pk primary key (list, item));
```

The primary key constraint will build an index that will make lookups
by list and item go quickly.

The SQL for operations on single items is straightforward:
See `data.go` for details.

### Batching for Performance

Where the SQL starts to get interesting is when we consider
tracking millions or billions of items. Hitting the database
for every individual item (and making a network round trip for
every item) will be crazy slow.

One of the oldest performance tricks in the book is to do
things in batches. PostgreSQL gives us lots of options to do just that.

The first thing to know is that sets of values can be repeated
in an insert statement. If we wanted to insert files `1.txt`
through `4.txt` into the list foo, we can do so in a single insert statement:

```
insert into lists
(list, item)
values
('foo', '1.txt'),
('foo', '2.txt'),
('foo', '3.txt'),
('foo', '4.txt');
```

Batch inserts are therefore possible.

Batch reads are also possible. Imagine a co-ordinator process
(or Goroutine) paging through a list of downloads and handing
off batches of downloads to workers.

I followed the advice at
[Use the Index, Luke](https://use-the-index-luke.com/blog/2013-07/pagination-done-the-postgresql-way)
to ensure that paging through a list of thousands of files performs
well.

Here's the SQL to get the first 1000 files

```
  select item,
         attempts
    from lists
   where list = 'foo'
order by list,
         item
   limit 1000;
```

Here's the SQL to get the next 1000 files, etc:

```
  select item,
         attempts
    from lists
   where list = 'foo'
     and item > '1000.txt'
order by list,
         item
   limit 1000;
```

We can also increment attempt counts in batches.

The first trick is to just add one to the current value in
the table, so that one bit of SQL can increment a list of
items whose attempt counts will vary:

```
set attempts = attempts + 1
```

The next trick is to provide the list of files as a single-columned
inline table in the `in` clause of the `update` statement useing the
[values](https://www.postgresql.org/docs/current/sql-values.html)
SQL statement.

```
update lists
   set attempts = attempts + 1
 where list = 'foo'
   and item in (values ('1.txt'),
                       ('2.txt'),
                       ('3.txt'),
                       ('4.txt'),
                       ('5.txt'));
```

The advantage of using a single-columned inline table via `values`
is that it's more efficient for PostgreSQL to process a table than
it would be a normal list of values in an `in` clause.
See [here](https://www.manniwood.com/2016_02_01/arrays_and_the_postgresql_query_planner.html)
and [here](https://dba.stackexchange.com/questions/91247/optimizing-a-postgres-query-with-a-large-in)
for details.

Deleting a batch of items from the table takes advantage of the same
performance optimization as the update above:

```
delete from lists
      where list = 'foo'
   and item in (values ('1.txt'),
                       ('2.txt'),
                       ('3.txt'),
                       ('4.txt'),
                       ('5.txt'));
```

This explains all of the SQL that is used in the `data.go` source
file. The SQL statements themselves are built up in a fairly straightforward
manner. More complex SQL might benefit from a templating system, whereas
here, introducing a templating system might be too much too early.

### Testing

The tests for `data.go` assume the existence/availability of a
PostgreSQL database that has been loaded with the required schema.

So really we are doing integration tests and not unit tests.

The upside of testing this way is knowing that the SQL code actually works,
because it uses an actual PostgreSQL database.

The downside of testing this way is the requirement of a PostgreSQL instance.
Using something like [sqlmock](https://github.com/DATA-DOG/go-sqlmock) would
allow actual unit testing.

## The REST API

Normally, the code in `data.go` would just live inside of a larger applicaiton,
but for learning purposes, I created a REST-like API to sit on top of
`data.go`.


The API for dealing with items in batches looks like this:

Add items `a.txt` through `i.txt` to the `downloads` list:

```
POST /iidy/v1/batch/lists/downloads -d '
a.txt
b.txt
c.txt
d.txt
e.txt
f.txt
g.txt
h.txt
i.txt
'
```

Get 1000 things from the `downloads` list, in alphabetical order,
starting from the beginning:

```
GET /iidy/v1/batch/lists/downloads?count=1000
```

Get the next 1000 items from the `downloads` list, starting after
the last item from the previous list (in this case, `z1000.txt`):

```
GET /iidy/v1/batch/lists/downloads?count=1000&after_id=z1000.txt
```

Normally, a worker would work on a batch of things, maintain an internal
list of failures, and update the failed items all in one call, like so:

```
POST /iidy/v1/batch/lists/downloads?action=increment -d '
b.txt
m.txt
z.txt
'
```

Successfully-processed items could be deleted from the list all
in one go, like so:

```
DELETE /iidy/v1/batch/lists/downloads -d '
a.txt
c.txt
d.txt
e.txt
...
'
```

The API for dealing with single items looks like this:

Get the number of attempts for `a.txt` from the list named `downloads`.

```
GET /iidy/v1/lists/downloads/a.txt
```

Add `a.txt` to the list named `downloads`.

```
POST /iidy/v1/lists/downloads/a.txt
```

Increment the number of attempts to process `a.txt` from the list named `downloads`.

```
POST /iidy/v1/lists/downloads/a.txt?action=increment
```

Delete `a.txt` from the list named `downloads`.

```
DELETE localhost:8080/iidy/v1/lists/downloads/a.txt
```

The API also speaks JSON, because JSON is an expected format for
REST services these days.

A plaintext API, as shown in these examples, is also used. This plaintext
API has the advantage of being able to work with the traditional suite
of command-line tools like `sed` and `awk`. But, of course, it has
the disadvantage of not being JSON, and a lot of services have come to
expect JSON as a default.

### API design considerations

REST is not really a standard, so there's a lot of lattitude in how to
design and implement a REST-like API.

Earlier iterations of IIDY used custom verbs for incrementing and doing
batch operations. So instead of `POST .../increment/...`, I used
`INCREMENT`. And instead of `DELETE .../batch/...`, I used `BULKDELETE`

However, I read
_Building Microservices in Go_, by Nic Jackson, who provided these guidelines:

 - POST creates a new resource or executes a controller
 - PUT updates (replaces?) a mutable resource
 - PATCH does a partial update of amutable resource
 - DELETE deletes a resource
 - HEAD is like GET that only returns headers and no body; it is used to see if a resource exists or not without the overhead of returning that resource's body

and that influenced the API that I have now, where I stick to the standard
HTTP verbs and stick to extra elemens in the URL to use IIDY's different
capabilities.

If this was written for a particular company rather than as a pet/learning
project, it could follow decisions/patterns that engeneering group had made
around how to design REST-like APIs. After all, REST is not a standard, so
whatever internal conventions the company already had would prevail.

### Error handling considerations

I followed the advice at https://blog.golang.org/go1.13-errors when
planning the error handling between data and the REST interface.

The pgx errors we will be dealing with are internal details to data.
To avoid exposing pgx errors to the REST part of the application,
I repackage them as new errors with the same text as the original errors.
I use the `%v` formatting verb, since `%w` would permit the caller to
unwrap the original pgx errors. I don't want to support pgx errors as part
of IIDY's API.

