package dto

import (
	"strconv"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

// Pagination limits applied when parsing query string parameters.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// PaginationFilter holds the parsed pagination parameters.
type PaginationFilter struct {
	Limit  int
	Cursor *domain.CursorToken
}

// DateRangeFilter holds optional from/to timestamps.
type DateRangeFilter struct {
	From *time.Time
	To   *time.Time
}

// ParsePagination parses "limit" and "cursor" query parameters.
// Returns false and writes an error to the context on invalid input.
func ParsePagination(c *gin.Context) (PaginationFilter, bool) {
	f := PaginationFilter{Limit: DefaultLimit}

	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxLimit {
			_ = c.Error(apierr.BadRequest("limit must be an integer between 1 and 100"))
			return f, false
		}
		f.Limit = v
	}

	if raw := c.Query("cursor"); raw != "" {
		ct, err := service.DecodeCursor(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid cursor"))
			return f, false
		}
		f.Cursor = ct
	}

	return f, true
}

// ParseDateRange parses "from" and "to" RFC 3339 query parameters and validates
// that from does not come after to. Returns false and writes an error on invalid input.
func ParseDateRange(c *gin.Context) (DateRangeFilter, bool) {
	var f DateRangeFilter

	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("from must be a valid ISO 8601 timestamp"))
			return f, false
		}
		f.From = &t
	}

	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("to must be a valid ISO 8601 timestamp"))
			return f, false
		}
		f.To = &t
	}

	if f.From != nil && f.To != nil && f.From.After(*f.To) {
		_ = c.Error(apierr.BadRequest("from must not be after to"))
		return f, false
	}

	return f, true
}

// ParseInt32 parses a named query parameter as an int32.
// Returns (nil, true) when the parameter is absent.
// Returns (nil, false) and writes an error on invalid input.
func ParseInt32(c *gin.Context, param string) (*int32, bool) {
	raw := c.Query(param)
	if raw == "" {
		return nil, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		_ = c.Error(apierr.BadRequest(param + " must be a valid integer"))
		return nil, false
	}
	i := int32(v)
	return &i, true
}

// ParseString returns a pointer to the query param value, or nil when absent.
func ParseString(c *gin.Context, param string) *string {
	raw := c.Query(param)
	if raw == "" {
		return nil
	}
	return &raw
}

// ParseStatus parses a "status" query parameter and validates it against a
// list of allowed values. Returns (nil, true) when absent, (nil, false) on
// invalid input.
func ParseStatus(c *gin.Context, allowed ...string) (*string, bool) {
	raw := c.Query("status")
	if raw == "" {
		return nil, true
	}
	for _, v := range allowed {
		if raw == v {
			return &raw, true
		}
	}
	_ = c.Error(apierr.BadRequest("status must be one of: " + joinStrings(allowed)))
	return nil, false
}

// OffsetPaginationFilter holds the parsed limit and offset parameters.
type OffsetPaginationFilter struct {
	Limit  int32
	Offset int32
}

// ParseOffsetPagination parses "limit" and "offset" query parameters.
// Returns false and writes an error to the context on invalid input.
func ParseOffsetPagination(c *gin.Context) (OffsetPaginationFilter, bool) {
	f := OffsetPaginationFilter{Limit: DefaultLimit}

	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > MaxLimit {
			_ = c.Error(apierr.BadRequest("limit must be an integer between 1 and 100"))
			return f, false
		}
		f.Limit = int32(v)
	}

	if raw := c.Query("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 0 {
			_ = c.Error(apierr.BadRequest("offset must be a non-negative integer"))
			return f, false
		}
		f.Offset = int32(v)
	}

	return f, true
}

// ToListParams converts an OffsetPaginationFilter to a domain.ListParams.
func (f OffsetPaginationFilter) ToListParams() domain.ListParams {
	return domain.ListParams{Limit: f.Limit, Offset: f.Offset}
}

// joinStrings joins a slice with ", " for use in error messages.
func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for _, s := range ss[1:] {
		out += ", " + s
	}
	return out
}
