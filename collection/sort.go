package collection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// sortKey is one compiled SearchOptions.Sort or ListOptions.Sort entry.
type sortKey struct {
	path       []string // "id", "title", or "data" followed by nested keys
	descending bool
	enum       []any // schema enum values in declared order, if any
}

// compileSortKeys parses keys of the form [-]id, [-]title, or [-]data.FIELD[.FIELD...]
// and attaches enum order from the collection schema.
func compileSortKeys(keys []string, schema json.RawMessage) ([]sortKey, error) {
	compiled := make([]sortKey, 0, len(keys))
	var root any
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
		if key.path[0] == "data" {
			if root == nil {
				var err error
				if root, err = jsonschema.UnmarshalJSON(bytes.NewReader(schema)); err != nil {
					return nil, err
				}
			}
			key.enum = fieldEnum(root, key.path[1:])
		}
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
			if order := compareKeyValues(key, x, y); order != 0 {
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

// compareKeyValues orders enum values by schema position, before values outside
// the enum, which use compareSortValues.
func compareKeyValues(key sortKey, x, y any) int {
	if key.enum != nil {
		ix, iy := enumIndex(key.enum, x), enumIndex(key.enum, y)
		if ix != iy {
			return ix - iy
		}
		if ix < len(key.enum) {
			return 0
		}
	}
	return compareSortValues(x, y)
}

func enumIndex(enum []any, value any) int {
	for i, candidate := range enum {
		if reflect.DeepEqual(candidate, value) {
			return i
		}
	}
	return len(enum)
}

// fieldEnum follows properties and local $ref pointers from the schema root to
// the field at path and returns its enum. Fields described only through
// combinators such as allOf or oneOf have no enum order.
func fieldEnum(root any, path []string) []any {
	node := root
	for _, segment := range path {
		properties, _ := schemaKeyword(root, node, "properties").(map[string]any)
		node = properties[segment]
	}
	enum, _ := schemaKeyword(root, node, "enum").([]any)
	return enum
}

// schemaKeyword returns keyword from node, following local "#/..." $ref
// pointers when node lacks it. Cycles and other reference forms yield nil.
func schemaKeyword(root, node any, keyword string) any {
	for range 32 {
		object, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		if value, ok := object[keyword]; ok {
			return value
		}
		ref, _ := object["$ref"].(string)
		pointer, ok := strings.CutPrefix(ref, "#")
		if !ok || (pointer != "" && !strings.HasPrefix(pointer, "/")) {
			return nil
		}
		node = root
		for _, token := range strings.Split(pointer, "/")[1:] {
			token = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
			child, _ := node.(map[string]any)
			node = child[token]
		}
	}
	return nil
}
