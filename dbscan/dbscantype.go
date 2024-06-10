package dbscan

func ScanAllType[T any](rows Rows) ([]T, error) {
	var results []T
	err := ScanAll(&results, rows)
	return results, err
}

func ScanOneType[T any](rows Rows) (T, error) {
	var result T
	err := ScanOne(&result, rows)
	return result, err
}

func ScanRowType[T any](rows Rows) (T, error) {
	var result T
	err := ScanRow(&result, rows)
	return result, err
}

func APIScanAllType[T any](api *API, rows Rows) ([]T, error) {
	var results []T
	err := api.ScanAll(&results, rows)
	return results, err
}

func APIScanOneType[T any](api *API, rows Rows) (T, error) {
	var result T
	err := api.ScanOne(&result, rows)
	return result, err
}

func APIScanRowType[T any](api *API, rows Rows) (T, error) {
	var result T
	err := api.ScanRow(&result, rows)
	return result, err
}
