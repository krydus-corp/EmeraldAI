package models

// Pagination constants
const (
	paginationDefaultLimit = 100
	paginationMaxLimit     = 1000
)

// PaginationReq holds pagination http fields and tags
type PaginationReq struct {
	Limit   int    `query:"limit" json:"limit"`
	Page    int    `query:"page" json:"page" validate:"min=0"`
	SortKey string `query:"sort_key" json:"sort_key" validate:"omitempty"`
	SortVal int    `query:"sort_val" json:"sort_val" validate:"omitempty,oneof=1 -1"`
}

// Transform checks and converts http pagination into database pagination model
func (p PaginationReq) Transform() Pagination {
	if p.Limit < 1 {
		p.Limit = paginationDefaultLimit
	}
	if p.Limit > paginationMaxLimit {
		p.Limit = paginationMaxLimit
	}

	sortKey, sortVal := "_id", -1
	if p.SortKey != "" {
		sortKey = p.SortKey
	}
	if p.SortVal != 0 {
		sortVal = p.SortVal
	}

	return Pagination{Limit: p.Limit, Offset: p.Page * p.Limit, SortKey: sortKey, SortVal: sortVal}
}

// Pagination data
type Pagination struct {
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	SortKey string `json:"sort_key,omitempty"`
	SortVal int    `json:"sort_val,omitempty"`
}
