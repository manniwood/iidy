/*
Package iidy is a PostgreSQL-backed checklist or "attempt list".

Sample Use Case

Let's say you want to download a million items, and you suspect not all
items will get successfully downloaded on the first attempt.

Start with a list of the items to be downloaded.

    connectionURL := "postgresql://iidy:password@localhost:5432/iidy?pool_max_conns=5&application_name=iidy"
    s, _ := iidy.NewPgStore(connectionURL)
    listName := "downloads"
    listItems := []string{"a.txt", "b.txt", "c.txt", "d.txt", "e.txt", "f.txt"}
    s.BulkAdd(context.Background(), listName, listItems)

A worker can get a certain number of items to work on:

    // gets "a.txt", "b.txt", "c.txt"
    items, _ := s.BulkGet(context.Background(), listName, "", 3)

For items that were unsuccessfully downloaded, the number of failed attempts
is incremented for that item. (A business rule can be set to abandon
downloading an item after a certain number of attempts.)

    count, _ := s.BulkInc(context.Background(), ListName, []string{"a.txt", "c.txt"})

Items that were successfully downloaded can be removed from the list.

    count, _ := s.BulkDel(context.Background(), ListName, []string{"b.txt"})

A worker can get more items from the list, starting past the last item in the
previously-worked-on batch:

    // gets "d.txt", "e.txt", "f.txt"
    items, _ := s.BulkGet(context.Background(), listName, "c.txt", 3)

And the cycle can continue.
*/
package iidy
