package pagination

import (
	"testing"
	"time"
)

func TestPaginationParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  PaginationParams
		wantErr bool
	}{
		{
			name: "valid params with defaults",
			params: PaginationParams{
				Limit:  10,
				Offset: 0,
				Page:   1,
			},
			wantErr: false,
		},
		{
			name: "valid params with custom values",
			params: PaginationParams{
				BaseRequestParams: BaseRequestParams{
					SortBy: "name",
					Order:  SortOrderAsc,
				},
				Limit:  50,
				Offset: 100,
				Page:   3,
			},
			wantErr: false,
		},
		{
			name: "limit too high",
			params: PaginationParams{
				Limit: 10001,
			},
			wantErr: true,
		},
		{
			name: "limit too low",
			params: PaginationParams{
				Limit: 0,
				Page:  1,
			},
			wantErr: false, // Should use default
		},
		{
			name: "negative offset",
			params: PaginationParams{
				Limit:  10,
				Offset: -1,
			},
			wantErr: true,
		},
		{
			name: "invalid page",
			params: PaginationParams{
				Limit: 10,
				Page:  0,
			},
			wantErr: false, // Should use default
		},
		{
			name: "invalid order",
			params: PaginationParams{
				BaseRequestParams: BaseRequestParams{
					Order: "invalid",
				},
				Limit: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("PaginationParams.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPaginationParams_GetOffset(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		want   int
	}{
		{
			name: "explicit offset",
			params: PaginationParams{
				Limit:  10,
				Offset: 20,
			},
			want: 20,
		},
		{
			name: "calculated from page",
			params: PaginationParams{
				Limit: 10,
				Page:  3,
			},
			want: 20,
		},
		{
			name: "first page",
			params: PaginationParams{
				Limit: 10,
				Page:  1,
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.params.GetOffset(); got != tt.want {
				t.Errorf("PaginationParams.GetOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaginationParams_GetPage(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		want   int
	}{
		{
			name: "explicit page",
			params: PaginationParams{
				Limit: 10,
				Page:  5,
			},
			want: 5,
		},
		{
			name: "calculated from offset",
			params: PaginationParams{
				Limit:  10,
				Offset: 20,
			},
			want: 3,
		},
		{
			name: "default page",
			params: PaginationParams{
				Limit: 10,
			},
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.params.GetPage(); got != tt.want {
				t.Errorf("PaginationParams.GetPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaginationParams_GetOrderClause(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		want   string
	}{
		{
			name: "ascending order",
			params: PaginationParams{
				BaseRequestParams: BaseRequestParams{
					SortBy: "name",
					Order:  SortOrderAsc,
				},
			},
			want: "name ASC",
		},
		{
			name: "descending order",
			params: PaginationParams{
				BaseRequestParams: BaseRequestParams{
					SortBy: "created_at",
					Order:  SortOrderDesc,
				},
			},
			want: "created_at DESC",
		},
		{
			name: "default values",
			params: PaginationParams{
				Limit: 10,
			},
			want: "created_at DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.params.GetOrderClause(); got != tt.want {
				t.Errorf("PaginationParams.GetOrderClause() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPageResponse(t *testing.T) {
	type User struct {
		ID   string
		Name string
	}

	users := []User{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
		{ID: "3", Name: "Charlie"},
	}

	params := &PaginationParams{
		Limit: 10,
		Page:  1,
	}

	resp := NewPageResponse(users, 25, params)

	if resp.Pagination == nil {
		t.Fatal("Pagination metadata is nil")
	}

	if resp.Pagination.Total != 25 {
		t.Errorf("Total = %v, want 25", resp.Pagination.Total)
	}

	if resp.Pagination.TotalPages != 3 {
		t.Errorf("TotalPages = %v, want 3", resp.Pagination.TotalPages)
	}

	if !resp.Pagination.HasNext {
		t.Error("HasNext should be true")
	}

	if resp.Pagination.HasPrev {
		t.Error("HasPrev should be false")
	}

	if len(resp.Data) != 3 {
		t.Errorf("Data length = %v, want 3", len(resp.Data))
	}
}

func TestCursorParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  CursorParams
		wantErr bool
	}{
		{
			name: "valid params",
			params: CursorParams{
				Limit:  10,
				Cursor: "eyJpZCI6IjEyMyJ9",
			},
			wantErr: false,
		},
		{
			name: "limit too high",
			params: CursorParams{
				Limit: 10001,
			},
			wantErr: true,
		},
		{
			name: "valid without cursor",
			params: CursorParams{
				Limit: 25,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("CursorParams.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCursorResponse(t *testing.T) {
	type User struct {
		ID   string
		Name string
	}

	users := []User{
		{ID: "1", Name: "Alice"},
		{ID: "2", Name: "Bob"},
	}

	params := &CursorParams{
		Limit: 10,
	}

	resp := NewCursorResponse(users, "next_cursor_123", "", params)

	if resp.Cursor == nil {
		t.Fatal("Cursor metadata is nil")
	}

	if resp.Cursor.NextCursor != "next_cursor_123" {
		t.Errorf("NextCursor = %v, want 'next_cursor_123'", resp.Cursor.NextCursor)
	}

	if !resp.Cursor.HasNext {
		t.Error("HasNext should be true")
	}

	if resp.Cursor.HasPrev {
		t.Error("HasPrev should be false")
	}

	if resp.Cursor.Count != 2 {
		t.Errorf("Count = %v, want 2", resp.Cursor.Count)
	}
}

func TestEncodeDecode_Cursor(t *testing.T) {
	id := "user123"
	ts := time.Now()
	value := "name_value"

	// Encode
	encoded, err := EncodeCursor(id, ts, value)
	if err != nil {
		t.Fatalf("EncodeCursor failed: %v", err)
	}

	if encoded == "" {
		t.Error("Encoded cursor is empty")
	}

	// Decode
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor failed: %v", err)
	}

	if decoded.ID != id {
		t.Errorf("Decoded ID = %v, want %v", decoded.ID, id)
	}

	if decoded.Value != value {
		t.Errorf("Decoded Value = %v, want %v", decoded.Value, value)
	}

	// Check timestamp (allow small difference due to precision)
	diff := decoded.Timestamp.Sub(ts)
	if diff > time.Millisecond || diff < -time.Millisecond {
		t.Errorf("Decoded Timestamp = %v, want %v", decoded.Timestamp, ts)
	}
}

func TestSimpleCursorEncodeDecode(t *testing.T) {
	original := "cursor_value_123"

	// Encode
	encoded := SimpleCursorEncode(original)
	if encoded == "" {
		t.Error("Encoded cursor is empty")
	}

	// Decode
	decoded, err := SimpleCursorDecode(encoded)
	if err != nil {
		t.Fatalf("SimpleCursorDecode failed: %v", err)
	}

	if decoded != original {
		t.Errorf("Decoded = %v, want %v", decoded, original)
	}
}

func TestEmptyResponses(t *testing.T) {
	type User struct {
		ID string
	}

	t.Run("empty page response", func(t *testing.T) {
		resp := NewEmptyPageResponse[User]()

		if len(resp.Data) != 0 {
			t.Errorf("Data length = %v, want 0", len(resp.Data))
		}

		if resp.Pagination.Total != 0 {
			t.Errorf("Total = %v, want 0", resp.Pagination.Total)
		}

		if resp.Pagination.HasNext {
			t.Error("HasNext should be false")
		}
	})

	t.Run("empty cursor response", func(t *testing.T) {
		resp := NewEmptyCursorResponse[User]()

		if len(resp.Data) != 0 {
			t.Errorf("Data length = %v, want 0", len(resp.Data))
		}

		if resp.Cursor.NextCursor != "" {
			t.Errorf("NextCursor = %v, want empty", resp.Cursor.NextCursor)
		}

		if resp.Cursor.HasNext {
			t.Error("HasNext should be false")
		}
	})
}

// Benchmark tests
func BenchmarkPaginationParams_Validate(b *testing.B) {
	params := PaginationParams{
		BaseRequestParams: BaseRequestParams{
			SortBy: "created_at",
			Order:  SortOrderDesc,
		},
		Limit:  10,
		Offset: 0,
		Page:   1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = params.Validate()
	}
}

func BenchmarkEncodeCursor(b *testing.B) {
	id := "user123"
	ts := time.Now()
	value := "test_value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = EncodeCursor(id, ts, value)
	}
}

func BenchmarkDecodeCursor(b *testing.B) {
	id := "user123"
	ts := time.Now()
	value := "test_value"
	encoded, _ := EncodeCursor(id, ts, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeCursor(encoded)
	}
}
