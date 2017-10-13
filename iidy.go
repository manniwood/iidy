package iidy

type Store interface {
	Add(listName string, itemID string) (err error)
	Get(listName string, itemID string) (attempts uint, ok bool, err error)
	Del(listName string, itemID string) (err error)
	Inc(listName string, itemID string) (err error)
}

type ListItem struct {
	ID       string
	Attempts uint
}

type BulkStore interface {
	Store
	BulkAdd(listName string, itemIDs []string) (err error)
	//BulkGet(listName string, startID string, count int) (listItems []ListItem, ok bool, err error)
	//BulkDel(listName string, itemIDs []string) (err error)
	//BulkInc(listName string, itemIDs []string) (err error)
}
