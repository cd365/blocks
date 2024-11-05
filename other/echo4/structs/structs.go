package structs

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// OfCount Count the total number of entries that meet the conditions.
type OfCount struct {
	IsCount bool  `json:"count" query:"count" form:"count" validate:"-"` // Whether the total number of entries is counted.
	Count   int64 `json:"-" query:"-" form:"-" validate:"-"`             // The total number of data records found.
}

// OfKeyword Search for keywords.
type OfKeyword struct {
	Keyword *string `json:"keyword" query:"keyword" form:"keyword" validate:"omitempty,min=1,max=32"` // Search for keywords [1,32]
}

func (s OfKeyword) GetKeyword() string {
	return fmt.Sprintf("%%%s%%", *s.Keyword)
}

// OfOrder Sort the retrieved data.
type OfOrder struct {
	Order *string `json:"order" query:"order" form:"order" validate:"omitempty,min=1,max=255"` // sort [1,255] StringToHex(field1:a,field2:d,field3:a...)
}

func (s OfOrder) GetOrder() string {
	if s.Order == nil || *s.Order == "" {
		return ""
	}
	*s.Order = strings.TrimSpace(*s.Order)
	if bts, err := hex.DecodeString(*s.Order); err == nil {
		return string(bts)
	}
	return *s.Order
}

// OfLimitOffset Control the data returned by the data list.
type OfLimitOffset struct {
	// Limit The number of entries of retrieved data. [1,1000]
	Limit *int64 `json:"limit" query:"limit" form:"limit" validate:"omitempty,min=1,max=1000"`

	// Offset Query offset. [0,)
	Offset *int64 `json:"offset" query:"offset" form:"offset" validate:"omitempty,min=0"`

	// Page Paginated page numbers. [1,)
	Page *int64 `json:"page" query:"page" form:"page" validate:"omitempty,min=1"`
}

func (s OfLimitOffset) GetLimit() int64 {
	if s.Limit == nil {
		return 1
	}
	return *s.Limit
}

func (s OfLimitOffset) GetOffset() int64 {
	if s.Page != nil {
		return (*s.Page - 1) * s.GetLimit()
	}
	if s.Offset != nil && *s.Offset >= 0 {
		return *s.Offset
	}
	return 0
}
