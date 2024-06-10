package dbscan

type RowScannerType[T any] struct {
	rs *RowScanner
}

func NewRowScannerType[T any](rows Rows) *RowScannerType[T] {
	return &RowScannerType[T]{rs: NewRowScanner(rows)}
}

func NewRowScannerTypeFromScanner[T any](rs *RowScanner) *RowScannerType[T] {
	return &RowScannerType[T]{rs: rs}
}

func (rs *RowScannerType[T]) Scan() (*T, error) {
	var result T
	err := rs.rs.Scan(&result)
	return &result, err
}
