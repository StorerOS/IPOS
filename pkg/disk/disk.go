package disk

type Info struct {
	Total  uint64
	Free   uint64
	Files  uint64
	Ffree  uint64
	FSType string

	Usage uint64
}
