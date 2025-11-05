package types

import "testing"

func TestDefaultPagination(t *testing.T) {
	p := DefaultPagination()
	if p.Page != 1 {
		t.Fatalf("expected Page=1, got %d", p.Page)
	}
	if p.PageSize != 20 {
		t.Fatalf("expected PageSize=20, got %d", p.PageSize)
	}
	if p.OrderBy != "created_at" {
		t.Fatalf("expected OrderBy=created_at, got %s", p.OrderBy)
	}
	if p.OrderDir != "desc" {
		t.Fatalf("expected OrderDir=desc, got %s", p.OrderDir)
	}
}
