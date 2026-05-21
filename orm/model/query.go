package model

type FilterType string

const (
	FilterTypePartialMatch FilterType = "partialMatch"
	FilterTypeExactMatch   FilterType = "exactMatch"
	FilterTypeDateGte      FilterType = "dateGte"
	FilterTypeDateLte      FilterType = "dateLte"
	FilterTypeGte          FilterType = "gte"
	FilterTypeLte          FilterType = "lte"
)

type Filter struct {
	Column string
	Value  any
	Type   FilterType
}

type SortDirection string

const (
	SortDirectionAsc  SortDirection = "asc"
	SortDirectionDesc SortDirection = "desc"
)

type Sorting struct {
	Column    string
	Direction SortDirection
}

type Pagination struct {
	Page  int
	Limit int
}

type SelectQuery struct {
	Filters    []Filter
	Sorting    []Sorting
	Entity     interface{}
	Pagination Pagination
}
