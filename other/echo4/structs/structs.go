package structs

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// OfCount 统计满足条件的总条数
type OfCount struct {
	IsCount bool  `json:"count" query:"count" form:"count" validate:"-"` // 是否统计总条数
	Count   int64 `json:"-" query:"-" form:"-" validate:"-"`             // 查到的数据总条数
}

// OfKeyword 搜索关键字
type OfKeyword struct {
	Keyword *string `json:"keyword" query:"keyword" form:"keyword" validate:"omitempty,min=1,max=32"` // 检索关键字 [1,32]
}

func (s OfKeyword) GetKeyword() string {
	return fmt.Sprintf("%%%s%%", *s.Keyword)
}

// OfOrder 检索数据排序
type OfOrder struct {
	Order *string `json:"order" query:"order" form:"order" validate:"omitempty,min=1,max=255"` // 排序 [1,255] 字符串转十六进制函数(field1:a,field2:d,field3:a...)
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

// OfLimitOffset 控制数据列表返回的数据
type OfLimitOffset struct {
	// Limit 检索数据的条数 [1,1000]
	Limit *int64 `json:"limit" query:"limit" form:"limit" validate:"omitempty,min=1,max=1000"`

	// Offset 已读数据的条数 [0,) (分页通过设置offset或viewed实现) offset 参数通用
	Offset *int64 `json:"offset" query:"offset" form:"offset" validate:"omitempty,min=0"`

	// Page 分页页码 [1,)
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
