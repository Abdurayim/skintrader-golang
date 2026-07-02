package pkg_test

import (
	"testing"
	"time"

	"skintrader-go/internal/pkg/pagination"
)

func TestNewOffsetParams_Defaults(t *testing.T) {
	p := pagination.NewOffsetParams(0, 0, 20, 100)
	if p.Page != 1 {
		t.Fatalf("expected page 1, got %d", p.Page)
	}
	if p.Limit != 20 {
		t.Fatalf("expected limit 20, got %d", p.Limit)
	}
}

func TestNewOffsetParams_MaxLimit(t *testing.T) {
	p := pagination.NewOffsetParams(1, 500, 20, 100)
	if p.Limit != 100 {
		t.Fatalf("expected capped limit 100, got %d", p.Limit)
	}
}

func TestNewOffsetParams_ValidValues(t *testing.T) {
	p := pagination.NewOffsetParams(3, 25, 20, 100)
	if p.Page != 3 {
		t.Fatalf("expected page 3, got %d", p.Page)
	}
	if p.Limit != 25 {
		t.Fatalf("expected limit 25, got %d", p.Limit)
	}
}

func TestOffsetParams_Offset(t *testing.T) {
	tests := []struct {
		page, limit, expected int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 10, 20},
		{5, 50, 200},
	}
	for _, tt := range tests {
		p := pagination.OffsetParams{Page: tt.page, Limit: tt.limit}
		if got := p.Offset(); got != tt.expected {
			t.Errorf("page=%d limit=%d: expected offset %d, got %d", tt.page, tt.limit, tt.expected, got)
		}
	}
}

func TestNewOffsetResult(t *testing.T) {
	p := pagination.OffsetParams{Page: 2, Limit: 10}
	r := pagination.NewOffsetResult(p, 35)

	if r.TotalItems != 35 {
		t.Fatalf("expected 35 total items, got %d", r.TotalItems)
	}
	if r.TotalPages != 4 {
		t.Fatalf("expected 4 total pages, got %d", r.TotalPages)
	}
	if !r.HasNext {
		t.Fatal("expected HasNext=true for page 2 of 4")
	}
	if !r.HasPrev {
		t.Fatal("expected HasPrev=true for page 2")
	}
}

func TestNewOffsetResult_FirstPage(t *testing.T) {
	p := pagination.OffsetParams{Page: 1, Limit: 10}
	r := pagination.NewOffsetResult(p, 25)

	if r.HasPrev {
		t.Fatal("first page should not have prev")
	}
	if !r.HasNext {
		t.Fatal("page 1 of 3 should have next")
	}
}

func TestNewOffsetResult_LastPage(t *testing.T) {
	p := pagination.OffsetParams{Page: 3, Limit: 10}
	r := pagination.NewOffsetResult(p, 25)

	if r.HasNext {
		t.Fatal("last page should not have next")
	}
	if !r.HasPrev {
		t.Fatal("page 3 should have prev")
	}
}

func TestNewOffsetResult_Empty(t *testing.T) {
	p := pagination.OffsetParams{Page: 1, Limit: 10}
	r := pagination.NewOffsetResult(p, 0)

	if r.TotalPages != 0 {
		t.Fatalf("expected 0 total pages, got %d", r.TotalPages)
	}
	if r.HasNext {
		t.Fatal("empty result should not have next")
	}
	if r.HasPrev {
		t.Fatal("empty result should not have prev")
	}
}

func TestCursorEncodeDecode(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	id := "test-id-123"

	encoded := pagination.EncodeCursor(now, id)
	if encoded == "" {
		t.Fatal("expected non-empty cursor")
	}

	decoded, err := pagination.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if decoded.ID != id {
		t.Fatalf("expected ID %s, got %s", id, decoded.ID)
	}
	if !decoded.CreatedAt.Equal(now) {
		t.Fatalf("expected time %v, got %v", now, decoded.CreatedAt)
	}
}

func TestDecodeCursor_Empty(t *testing.T) {
	decoded, err := pagination.DecodeCursor("")
	if err != nil {
		t.Fatalf("expected no error for empty cursor, got: %v", err)
	}
	if decoded != nil {
		t.Fatal("expected nil cursor for empty string")
	}
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := pagination.DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecodeCursor_InvalidJSON(t *testing.T) {
	// Valid base64 but invalid JSON
	_, err := pagination.DecodeCursor("bm90LWpzb24=")
	if err == nil {
		t.Fatal("expected error for invalid JSON cursor")
	}
}

func TestNewCursorParams(t *testing.T) {
	p := pagination.NewCursorParams("cursor123", 50, 20, 100)
	if p.Cursor != "cursor123" {
		t.Fatalf("expected cursor cursor123, got %s", p.Cursor)
	}
	if p.Limit != 50 {
		t.Fatalf("expected limit 50, got %d", p.Limit)
	}
}

func TestNewCursorParams_Defaults(t *testing.T) {
	p := pagination.NewCursorParams("", 0, 20, 100)
	if p.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", p.Limit)
	}
}

func TestNewCursorParams_MaxLimit(t *testing.T) {
	p := pagination.NewCursorParams("", 500, 20, 100)
	if p.Limit != 100 {
		t.Fatalf("expected capped limit 100, got %d", p.Limit)
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		input    string
		def      int
		expected int
	}{
		{"", 1, 1},
		{"5", 1, 5},
		{"0", 1, 1},
		{"-1", 1, 1},
		{"abc", 1, 1},
		{"100", 1, 100},
	}
	for _, tt := range tests {
		got := pagination.ParsePage(tt.input, tt.def)
		if got != tt.expected {
			t.Errorf("ParsePage(%q, %d) = %d, want %d", tt.input, tt.def, got, tt.expected)
		}
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		input    string
		def, max int
		expected int
	}{
		{"", 20, 100, 20},
		{"50", 20, 100, 50},
		{"0", 20, 100, 20},
		{"-5", 20, 100, 20},
		{"abc", 20, 100, 20},
		{"200", 20, 100, 100},
	}
	for _, tt := range tests {
		got := pagination.ParseLimit(tt.input, tt.def, tt.max)
		if got != tt.expected {
			t.Errorf("ParseLimit(%q, %d, %d) = %d, want %d", tt.input, tt.def, tt.max, got, tt.expected)
		}
	}
}
