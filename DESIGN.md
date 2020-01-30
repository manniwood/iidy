# Notes on the design of IIDY

To 

  // TODO: put this in the README instead
  // REST is not a standard, but we will take inspiration from the book
  // _How to Build Microservices in Go_:
  // POST creates a new resource or executes a controller
  // PUT updates (replaces?) a mutable resource
  // PATCH does a partial update of amutable resource
  // DELETE deletes a resource, though for us, would delete delete a whole list?
  // HEAD is like GET that only returns headers and no body; used to see if a resource exists or not without 

  // TODO: HEAD /v1/lists/<listname>
  // return 200 if list exists

integ test vs. unit test

// NOTE on error handling: we follow the advice at https://blog.golang.org/go1.13-errors:
// The pgx errors we will be dealing with are internal details.
// To avoid exposing them to the caller, we repackage them as new
// errors with the same text. We use the %v formatting verb, since
// %w would permit the caller to unwrap the original pgx errors.
// We don't want to support pgx errors as part of our API.

