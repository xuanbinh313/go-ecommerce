package models

import (
	"encoding/json"
	"time"
)

type Currency struct {
	Code          string      `json:"code"`
	Symbol        string      `json:"symbol"`
	SymbolNative  string      `json:"symbol_native"`
	Name          string      `json:"name"`
	DecimalDigits int         `json:"decimal_digits"`
	Rounding      json.Number `json:"rounding"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}
