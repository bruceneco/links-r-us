package repository

// Iterator is a contract to pass along a set of items orchestrating its progress.
type Iterator interface {
	// Next advances the iterator. If no more items are available or an error occurs, calls to Next() return false.
	Next() bool
	// Error returns the last error encountered by the Iterator.
	Error() error
	// Close releases any resources associated with an Iterator.
	Close() error
}
