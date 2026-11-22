package collection

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"golang.org/x/text/message"
)

// RefKeyword marks schema locations whose values are IDs of documents in the
// same collection. Writes check the ID syntax; Doctor reports missing targets.
const RefKeyword = "x-means-ref"

// reference is one document ID found at a data location marked by RefKeyword.
type reference struct {
	location []string
	id       string
}

type refExt struct {
	found func(reference)
}

func (r *refExt) Validate(ctx *jsonschema.ValidatorContext, v any) {
	id, ok := v.(string)
	if !ok || validateDocumentID(id) != nil {
		ctx.AddError(&invalidRef{})
		return
	}
	if r.found != nil {
		r.found(reference{location: append([]string(nil), ctx.ValueLocation()...), id: id})
	}
}

type invalidRef struct{}

func (*invalidRef) KeywordPath() []string { return []string{RefKeyword} }

func (*invalidRef) LocalizedString(*message.Printer) string {
	return "must be a document ID string"
}

// refVocabulary registers RefKeyword. found receives each syntactically valid
// reference during validation; uses records whether any schema enables it.
func refVocabulary(found func(reference), uses *bool) (*jsonschema.Vocabulary, error) {
	url := "https://github.com/hmans/means/vocab/ref"
	meta, err := jsonschema.UnmarshalJSON(strings.NewReader(`{"properties":{"` + RefKeyword + `":{"type":"boolean"}}}`))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(url, meta); err != nil {
		return nil, err
	}
	schema, err := compiler.Compile(url)
	if err != nil {
		return nil, err
	}
	return &jsonschema.Vocabulary{
		URL:    url,
		Schema: schema,
		Compile: func(_ *jsonschema.CompilerContext, object map[string]any) (jsonschema.SchemaExt, error) {
			if enabled, _ := object[RefKeyword].(bool); !enabled {
				return nil, nil
			}
			*uses = true
			return &refExt{found: found}, nil
		},
	}, nil
}

// compileSchema compiles a self-contained schema document with the means
// vocabulary. found may be nil when references need not be collected.
func compileSchema(location string, doc any, found func(reference)) (*jsonschema.Schema, bool, error) {
	var uses bool
	vocabulary, err := refVocabulary(found, &uses)
	if err != nil {
		return nil, false, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(selfContainedLoader{})
	compiler.RegisterVocabulary(vocabulary)
	compiler.AssertVocabs()
	if err := compiler.AddResource(location, doc); err != nil {
		return nil, false, fmt.Errorf("register JSON Schema: %w", err)
	}
	validator, err := compiler.Compile(location)
	if err != nil {
		return nil, false, fmt.Errorf("invalid JSON Schema: %w", err)
	}
	return validator, uses, nil
}

// references returns a function listing the reference values in data, or nil
// when the loaded schema does not use RefKeyword. The function is not safe for
// concurrent use.
func (c *Collection) references() (func(map[string]any) []reference, error) {
	if !c.usesRefs {
		return nil, nil
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(c.schema))
	if err != nil {
		return nil, err
	}
	var found []reference
	validator, _, err := compileSchema(filepath.Join(c.path, ConfigFilename), doc, func(ref reference) { found = append(found, ref) })
	if err != nil {
		return nil, err
	}
	return func(data map[string]any) []reference {
		found = nil
		_ = validator.Validate(data)
		return found
	}, nil
}
