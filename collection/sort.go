package collection

import (
	"encoding/json"
	"fmt"
	"math/big"
	"slices"
	"strings"
)

// sortKey is one compiled SearchOptions.Sort or ListOptions.Sort entry.
type sortKey struct {
	path       []string // "id", "title", or "data" followed by nested keys
	descending bool
}

// compileSortKeys parses keys of the form [-]id, [-]title, or [-]data.FIELD[.FIELD...].
func compileSortKeys(keys []string) ([]sortKey, error) {
	compiled := make([]sortKey, 0, len(keys))
	seen := map[string]bool{}
	for _, raw := range keys {
		key := sortKey{}
		name, descending := strings.CutPrefix(raw, "-")
		key.descending = descending
		key.path = strings.Split(name, ".")
		valid := false
		switch key.path[0] {
		case "id", "title":
			valid = len(key.path) == 1
		case "data":
			valid = len(key.path) > 1 && !slices.Contains(key.path[1:], "")
		}
		if !valid {
			return nil, fmt.Errorf("invalid sort key %q: use id, title, or data.FIELD (dots address nested fields), with - for descending", raw)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate sort key %q", name)
		}
		seen[name] = true
		compiled = append(compiled, key)
	}
	return compiled, nil
}

// sortValue returns the frontmatter value addressed by path. Missing fields,
// nulls, and paths through non-objects report false.
func sortValue(frontmatter map[string]any, path []string) (any, bool) {
	var value any = frontmatter
	for _, segment := range path {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		if value, ok = object[segment]; !ok {
			return nil, false
		}
	}
	return value, value != nil
}

// sortHits orders hits by keys. Missing values sort last in either direction.
// Ties fall back to the existing order when ranked (relevance, then ID) and to
// ID otherwise, so the result is always deterministic.
func sortHits(hits []SearchHit, keys []sortKey, ranked bool) {
	if len(keys) == 0 {
		return
	}
	slices.SortStableFunc(hits, func(a, b SearchHit) int {
		for _, key := range keys {
			x, xOK := sortValue(a.Frontmatter, key.path)
			y, yOK := sortValue(b.Frontmatter, key.path)
			switch {
			case !xOK && !yOK:
				continue
			case !xOK:
				return 1
			case !yOK:
				return -1
			}
			if order := compareSortValues(x, y); order != 0 {
				if key.descending {
					return -order
				}
				return order
			}
		}
		if ranked {
			return 0
		}
		return strings.Compare(a.ID, b.ID)
	})
}

// sortTypeRank orders mixed JSON types: booleans, numbers, strings, arrays, objects.
func sortTypeRank(value any) int {
	switch value.(type) {
	case bool:
		return 0
	case json.Number:
		return 1
	case string:
		return 2
	case []any:
		return 3
	default:
		return 4
	}
}

// compareSortValues compares two non-null values. Booleans order false before
// true, numbers compare exactly, and strings compare by Unicode code points.
// Arrays and objects compare equal among themselves.
func compareSortValues(x, y any) int {
	if rx, ry := sortTypeRank(x), sortTypeRank(y); rx != ry {
		return rx - ry
	}
	switch x := x.(type) {
	case bool:
		switch y := y.(bool); {
		case x == y:
			return 0
		case !x:
			return -1
		default:
			return 1
		}
	case json.Number:
		a, aOK := new(big.Rat).SetString(x.String())
		b, bOK := new(big.Rat).SetString(y.(json.Number).String())
		if aOK && bOK {
			return a.Cmp(b)
		}
		return strings.Compare(x.String(), y.(json.Number).String())
	case string:
		return strings.Compare(x, y.(string))
	}
	return 0
}
