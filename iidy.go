package iidy

type Iddy interface {
	Add(listName string, itemID string) (err error)
	Get(listName string, itemID string) (attempts uint, ok bool, err error)
	Del(listName string, itemID string) (err error)
	Inc(listName string, itemID string) (err error)
}
