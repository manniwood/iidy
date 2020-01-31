# Notes on the design of IIDY

IIDY is a pet project where I explore API/service ideas in Go.

## Constraint

How much can I get done using just the Go standard library?
IIDY needs a library to connect to PostgreSQL, but otherwise,
it uses the standard library.

## IIDY Purpose

I've worked in jobs where there would be millions of things to
be processed, such as downloading a bunch of files for an organization.
Naturally, some of the things wouldn't work on the first try (though
most would).

A certain pattern emerged:

1) Get the list of things to be processed, and put the names of those
things in a list.

2) Process those things, and, when successful, delete them from the list;
when failed, increment the number of attempts for those failed items
in the list.

3) Try failed items again later. Abandon items that failed to be
processed after a certain number of attempts, by deleting them
from the list.

4) Eventually, the list of items to be processed ends up empty.

A variation on this, for really large lists, is to maybe have a co-ordinator
process get large batches of things to work on from the list, hand
those items off to other workers and have those workers report back
to the list, by either deleting items from the list that were
successfully processed, or incrementing the number of attempts
for failed items.

## API

The REST-like interface for this looks something like this, for
a list of items that need to be downloaded (named downloads here):

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

Of course, if we are deailing will millions of items, going to a REST endpoint
for individual items is really inefficient. So, of course, a more realistic
API (from the point of view of efficiency) should support batching.

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

### API design considerations

Earlier iterations of IIDY used custom verbs for incrementing and doing
batch operations. So instead of `POST ...?action=increment`, I had
`INCREMENT`. And instead of `DELETE .../batch/...`, I had `BULKDELETE`

However, I read
_Building Microservices in Go_, by Nic Jackson, who provided these guidelines:

 - POST creates a new resource or executes a controller
 - PUT updates (replaces?) a mutable resource
 - PATCH does a partial update of amutable resource
 - DELETE deletes a resource
 - HEAD is like GET that only returns headers and no body; it is used to see if a resource exists or not without the overhead of returning that resource's body

And that influenced the API that I have now.

integ test vs. unit test

// NOTE on error handling: we follow the advice at https://blog.golang.org/go1.13-errors:
// The pgx errors we will be dealing with are internal details.
// To avoid exposing them to the caller, we repackage them as new
// errors with the same text. We use the %v formatting verb, since
// %w would permit the caller to unwrap the original pgx errors.
// We don't want to support pgx errors as part of our API.

