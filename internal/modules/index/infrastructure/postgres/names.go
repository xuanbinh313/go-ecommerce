package postgres

import (
	"strconv"
	"strings"
	"unicode/utf16"
)

// EntityPartitionName and RelationPartitionName follow Medusa's
// normalizeTableName/getPivotTableName naming rules.
func EntityPartitionName(entity string) string { return "cat_" + normalizeTableName(entity) }
func RelationPartitionName(parentEntity, childEntity string) (table, pivot string) {
	return "cat_pivot_" + normalizeTableName(parentEntity+childEntity), parentEntity + "-" + childEntity
}

func normalizeTableName(name string) string {
	compressed := compressName(strings.ToLower(name), 58)
	var out strings.Builder
	for _, r := range compressed {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' {
			out.WriteRune(r)
		} else {
			out.WriteByte('_')
		}
	}
	return out.String()
}

func compressName(name string, limit int) string {
	if len(name) <= limit {
		return strings.ToLower(name)
	}
	hash := simpleHash(name)
	parts := strings.Split(strings.ToLower(name), "_")
	dedup := make([]string, 0, len(parts))
	for i, part := range parts {
		if i == 0 || part != parts[i-1] {
			dedup = append(dedup, part)
		}
	}
	result := strings.Join(dedup, "_")
	if len(result) > limit {
		parts = strings.Split(camelToSnake(name), "_")
		for i, part := range parts {
			if len(part) > 4 {
				parts[i] = part[:4]
			}
		}
		result = strings.Join(parts, "_")
	}
	diff := len(result) + len(hash) - limit
	if diff > 0 {
		return strings.ToLower(result[:len(result)-diff] + hash)
	}
	return strings.ToLower(result + hash)
}

func simpleHash(value string) string {
	hash := int64(5381)
	for _, unit := range utf16.Encode([]rune(value)) {
		shifted := int32(hash) << 5
		hash = int64(shifted) + hash + int64(unit)
	}
	return strconv.FormatInt(hash, 16)
}

func camelToSnake(value string) string {
	var out strings.Builder
	for i, r := range value {
		if r >= 'A' && r <= 'Z' && i > 0 {
			prev := rune(value[i-1])
			if prev >= 'a' && prev <= 'z' {
				out.WriteByte('_')
			}
		}
		out.WriteRune(r)
	}
	return strings.ToLower(out.String())
}
