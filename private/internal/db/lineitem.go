package db

import (
	"time"

	"gorm.io/datatypes"
)

type LineItemKind = uint8

const (
	DebitLineItemKind LineItemKind = iota
	CreditLineItemKind
)

type LineItem struct {
	DateTime time.Time      `json:"dateTime" db:"dateTime" gorm:"column:dateTime;primaryKey"`
	Payer    string         `json:"payer"    db:"payer"    gorm:"column:payer;primaryKey;index:payer"`
	Payee    string         `json:"payee"    db:"payee"    gorm:"column:payee;primaryKey;index:payee"`
	URN      string         `json:"urn"      db:"urn"      gorm:"column:urn;primaryKey;index:urn"`
	Kind     LineItemKind   `json:"kind"     db:"kind"     gorm:"column:kind;index:kind"`
	Usage    int64          `json:"usage"    db:"usage"    gorm:"column:usage"`
	Detail   datatypes.JSON `json:"detail"   db:"detail"   gorm:"column:detail"`
	Labels   datatypes.JSON `json:"labels"   db:"labels"   gorm:"column:labels;index:labels"`
}
