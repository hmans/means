package collection

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// encodeMarkdown writes the order keys first, in that order; other top-level
// keys follow alphabetically.
func encodeMarkdown(frontmatter map[string]any, body string, order ...string) (string, error) {
	if !utf8.ValidString(body) {
		return "", fmt.Errorf("body must be valid UTF-8")
	}
	data, err := json.Marshal(frontmatter)
	if err != nil {
		return "", fmt.Errorf("encode frontmatter: %w", err)
	}
	// Parse JSON directly into YAML nodes to preserve strings and exact numbers.
	var node yaml.Node
	if err := yaml.Unmarshal(escapeYAMLControls(data), &node); err != nil {
		return "", err
	}
	if len(order) > 0 {
		mapping := node.Content[0]
		rank := func(i int) int {
			if r := slices.Index(order, mapping.Content[2*i].Value); r >= 0 {
				return r
			}
			return len(order)
		}
		pairs := make([]int, len(mapping.Content)/2)
		for i := range pairs {
			pairs[i] = i
		}
		slices.SortStableFunc(pairs, func(a, b int) int { return rank(a) - rank(b) })
		content := make([]*yaml.Node, 0, len(mapping.Content))
		for _, i := range pairs {
			content = append(content, mapping.Content[2*i], mapping.Content[2*i+1])
		}
		mapping.Content = content
	}
	var blockStyle func(*yaml.Node)
	blockStyle = func(n *yaml.Node) {
		n.Style = 0
		// The encoder leaves a plain << unquoted, which YAML reads as a merge key,
		// folds the non-ASCII line breaks NEL, LS, and PS in other styles, and
		// writes block scalars with a leading tab that it cannot parse.
		if n.Kind == yaml.ScalarNode && n.Tag == "!!str" && (n.Value == "<<" || strings.HasPrefix(n.Value, "\t") || strings.ContainsAny(n.Value, "\u0085\u2028\u2029")) {
			n.Style = yaml.DoubleQuotedStyle
		}
		for _, child := range n.Content {
			blockStyle(child)
		}
	}
	blockStyle(&node)
	var out bytes.Buffer
	out.WriteString("---\n")
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return "", fmt.Errorf("encode YAML frontmatter: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	out.WriteString("---\n")
	out.WriteString(body)
	// Never write frontmatter that means itself would read back differently.
	stored, _, err := parseMarkdown(out.Bytes())
	if err != nil {
		return "", fmt.Errorf("encoded frontmatter does not round-trip: %w", err)
	}
	want, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("encode frontmatter: %w", err)
	}
	if path, ok := firstDifference(want, stored, "frontmatter"); ok {
		return "", fmt.Errorf("%s cannot be stored as YAML without changing its value; numbers must fit the float64 range", path)
	}
	return out.String(), nil
}

// escapeYAMLControls escapes the characters JSON permits raw inside strings
// but a YAML stream rejects, so they reach the encoder, which escapes them.
func escapeYAMLControls(data []byte) []byte {
	var out bytes.Buffer
	for _, r := range string(data) {
		if r == 0x7f || (r >= 0x80 && r <= 0x9f) || r == 0xfffe || r == 0xffff {
			fmt.Fprintf(&out, `\u%04x`, r)
		} else {
			out.WriteRune(r)
		}
	}
	return out.Bytes()
}

// firstDifference returns the path of the first value that differs between two
// decoded JSON values. Numbers compare by their exact text.
func firstDifference(want, got any, path string) (string, bool) {
	switch want := want.(type) {
	case map[string]any:
		got, ok := got.(map[string]any)
		if !ok || len(got) != len(want) {
			return path, true
		}
		for _, key := range slices.Sorted(maps.Keys(want)) {
			value, exists := got[key]
			if !exists {
				return fmt.Sprintf("%s key %q", path, key), true
			}
			if at, differs := firstDifference(want[key], value, path+"."+key); differs {
				return at, true
			}
		}
		return "", false
	case []any:
		got, ok := got.([]any)
		if !ok || len(got) != len(want) {
			return path, true
		}
		for i := range want {
			if at, differs := firstDifference(want[i], got[i], fmt.Sprintf("%s[%d]", path, i)); differs {
				return at, true
			}
		}
		return "", false
	}
	return path, want != got
}

func parseMarkdown(data []byte) (map[string]any, string, error) {
	if !utf8.Valid(data) {
		return nil, "", fmt.Errorf("document must be valid UTF-8")
	}
	first, rest, found := bytes.Cut(data, []byte("\n"))
	if !found || string(bytes.TrimSuffix(first, []byte("\r"))) != "---" {
		return nil, "", fmt.Errorf("document must start with a --- frontmatter delimiter")
	}
	header := rest
	for offset := 0; offset < len(header); {
		line, remaining, newline := bytes.Cut(header[offset:], []byte("\n"))
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			var node yaml.Node
			decoder := yaml.NewDecoder(bytes.NewReader(header[:offset]))
			if err := decoder.Decode(&node); err != nil {
				return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
			}
			var extra yaml.Node
			if err := decoder.Decode(&extra); err != io.EOF {
				return nil, "", fmt.Errorf("frontmatter must contain exactly one YAML mapping")
			}
			value, err := yamlValue(node.Content[0])
			if err != nil {
				return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
			}
			frontmatter, ok := value.(map[string]any)
			if !ok {
				return nil, "", fmt.Errorf("frontmatter must be an object")
			}
			return frontmatter, string(remaining), nil
		}
		if !newline {
			break
		}
		offset += len(line) + 1
	}
	return nil, "", fmt.Errorf("missing closing --- frontmatter delimiter")
}

// Convert the JSON-compatible subset of YAML without rounding decimal numbers
// or converting plain dates into time.Time values. Aliases and merge keys are
// deliberately unsupported so field values remain explicit.
func yamlValue(n *yaml.Node) (any, error) {
	if (n.Kind == yaml.MappingNode && n.Tag != "!!map") || (n.Kind == yaml.SequenceNode && n.Tag != "!!seq") {
		return nil, fmt.Errorf("line %d: custom YAML tags are not supported", n.Line)
	}
	switch n.Kind {
	case yaml.MappingNode:
		result := make(map[string]any, len(n.Content)/2)
		for i := 0; i < len(n.Content); i += 2 {
			key := n.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return nil, fmt.Errorf("line %d: object keys must be strings", key.Line)
			}
			if _, exists := result[key.Value]; exists {
				return nil, fmt.Errorf("line %d: duplicate field %q", key.Line, key.Value)
			}
			value, err := yamlValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			result[key.Value] = value
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, 0, len(n.Content))
		for _, child := range n.Content {
			value, err := yamlValue(child)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		return result, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str", "!!timestamp":
			return n.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool":
			var value bool
			err := n.Decode(&value)
			return value, err
		case "!!int", "!!float":
			if value, err := jsonschema.UnmarshalJSON(strings.NewReader(n.Value)); err == nil {
				if number, ok := value.(json.Number); ok {
					return number, nil
				}
			}
			// Normalize YAML numeric forms such as hexadecimal to JSON numbers.
			var value any
			if err := n.Decode(&value); err != nil {
				return nil, err
			}
			encoded, err := json.Marshal(value)
			if err == nil {
				return jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
			}
		}
	}
	return nil, fmt.Errorf("line %d: unsupported YAML value; use JSON-compatible values without aliases, merge keys, custom tags, or non-finite numbers", n.Line)
}
