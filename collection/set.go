package collection

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// FieldOp selects how a FieldEdit changes its target.
type FieldOp int

const (
	// FieldAssign sets the value, creating missing parent objects. A nil Value
	// deletes the field.
	FieldAssign FieldOp = iota
	// FieldAdd appends Value to an array unless an equal element exists. A
	// missing field becomes a one-element array.
	FieldAdd
	// FieldRemove removes every element equal to Value from an array. A missing
	// field is left missing.
	FieldRemove
)

// FieldEdit changes one frontmatter field. Path starts at the frontmatter and
// must begin with "title" or "data", for example {"data", "status"}.
type FieldEdit struct {
	Path  []string
	Op    FieldOp
	Value any
}

// Set applies field edits in order to the latest stored document. An empty hash
// edits the latest version; otherwise use the current hash from Read. Set shares
// Append's edit lock, validation, embedding maintenance, and conflict checks.
// Edits replace metadata as a whole, so a nil assigned value deletes a field
// rather than storing null. The existing filename is kept.
func (c *Collection) Set(ctx context.Context, id, hash string, edits ...FieldEdit) (*Document, error) {
	if len(edits) == 0 {
		return nil, fmt.Errorf("supply at least one field edit")
	}
	return c.modify(ctx, id, hash, func(current *Document) (Patch, bool, error) {
		frontmatter, err := normalizeObject(current.Frontmatter)
		if err != nil {
			return Patch{}, false, err
		}
		delete(frontmatter, "id")
		for _, edit := range edits {
			if err := applyFieldEdit(frontmatter, edit); err != nil {
				return Patch{}, false, err
			}
		}
		return Patch{Frontmatter: frontmatter}, true, nil
	})
}

func applyFieldEdit(frontmatter map[string]any, edit FieldEdit) error {
	name := strings.Join(edit.Path, ".")
	if len(edit.Path) == 0 || (edit.Path[0] != "title" && edit.Path[0] != "data") {
		return fmt.Errorf("field path %q must start with title or data", name)
	}
	if edit.Path[0] == "title" && len(edit.Path) > 1 {
		return fmt.Errorf("title has no nested fields")
	}
	for _, key := range edit.Path {
		if key == "" {
			return fmt.Errorf("field path %q contains an empty name", name)
		}
	}
	value := edit.Value
	if value != nil {
		normalized, err := normalizeObject(map[string]any{"v": value})
		if err != nil {
			return err
		}
		value = normalized["v"]
	}
	parent := frontmatter
	for _, key := range edit.Path[:len(edit.Path)-1] {
		next, exists := parent[key]
		if !exists {
			if edit.Op != FieldAssign || value == nil {
				return nil
			}
			child := map[string]any{}
			parent[key] = child
			parent = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("cannot edit %q: %q is not an object", name, key)
		}
		parent = child
	}
	key := edit.Path[len(edit.Path)-1]
	switch edit.Op {
	case FieldAssign:
		if value == nil {
			delete(parent, key)
		} else {
			parent[key] = value
		}
		return nil
	case FieldAdd, FieldRemove:
		if value == nil {
			return fmt.Errorf("cannot add or remove null in %q", name)
		}
		existing, exists := parent[key]
		if !exists {
			if edit.Op == FieldAdd {
				parent[key] = []any{value}
			}
			return nil
		}
		items, ok := existing.([]any)
		if !ok {
			return fmt.Errorf("cannot add to or remove from %q: it is not an array", name)
		}
		result := make([]any, 0, len(items)+1)
		found := false
		for _, item := range items {
			if reflect.DeepEqual(item, value) {
				found = true
				if edit.Op == FieldRemove {
					continue
				}
			}
			result = append(result, item)
		}
		if edit.Op == FieldAdd && !found {
			result = append(result, value)
		}
		parent[key] = result
		return nil
	default:
		return fmt.Errorf("unknown field operation %d", edit.Op)
	}
}
