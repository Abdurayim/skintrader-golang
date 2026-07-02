package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// OffsetParams for admin panel queries with page numbers.
type OffsetParams struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// OffsetResult contains offset pagination metadata.
type OffsetResult struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
	HasNext    bool  `json:"hasNext"`
	HasPrev    bool  `json:"hasPrev"`
}

// CursorParams for client API queries.
type CursorParams struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

// CursorResult contains cursor pagination metadata.
type CursorResult struct {
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
	Limit      int    `json:"limit"`
}

// Cursor encodes a timestamp + ID for keyset pagination.
type Cursor struct {
	CreatedAt time.Time `json:"t"`
	ID        string    `json:"i"`
}

func NewOffsetParams(page, limit, defaultLimit, maxLimit int) OffsetParams {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return OffsetParams{Page: page, Limit: limit}
}

func (p OffsetParams) Offset() int {
	return (p.Page - 1) * p.Limit
}

func NewOffsetResult(params OffsetParams, totalItems int64) OffsetResult {
	totalPages := int(totalItems) / params.Limit
	if int(totalItems)%params.Limit > 0 {
		totalPages++
	}
	return OffsetResult{
		Page:       params.Page,
		Limit:      params.Limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

func NewCursorParams(cursor string, limit, defaultLimit, maxLimit int) CursorParams {
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return CursorParams{Cursor: cursor, Limit: limit}
}

func EncodeCursor(createdAt time.Time, id string) string {
	c := Cursor{CreatedAt: createdAt, ID: id}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

func DecodeCursor(cursor string) (*Cursor, error) {
	if cursor == "" {
		return nil, nil
	}
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor format")
	}
	var c Cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid cursor data")
	}
	return &c, nil
}

func ParsePage(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}

func ParseLimit(s string, def, max int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	if v > max {
		return max
	}
	return v
}
