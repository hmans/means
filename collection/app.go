package collection

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// AppConfig describes optional collection views. Expressions are compiled by the
// TUI, not by collection operations or MCP.
type AppConfig struct {
	Views []AppView `json:"views,omitempty"`
}

type AppView struct {
	Title   string   `json:"title"`
	Filter  string   `json:"filter,omitempty"`
	Columns []string `json:"columns,omitempty"`
}

func parseApp(value any) (AppConfig, error) {
	var app AppConfig
	object, ok := value.(map[string]any)
	if !ok {
		return app, fmt.Errorf("app must be an object")
	}
	encoded, _ := json.Marshal(object)
	if len(encoded) > 65536 {
		return app, fmt.Errorf("app exceeds 64 KiB")
	}
	for key := range object {
		if key != "views" {
			return app, fmt.Errorf("unknown app field %q", key)
		}
	}
	if _, exists := object["views"]; !exists {
		return app, nil
	}
	views, ok := object["views"].([]any)
	if !ok || len(views) > 32 {
		return app, fmt.Errorf("app.views must be an array of at most 32 views")
	}
	seen := map[string]bool{"All documents": true}
	for i, raw := range views {
		fields, ok := raw.(map[string]any)
		if !ok {
			return app, fmt.Errorf("app.views[%d] must be an object", i)
		}
		for key := range fields {
			if key != "title" && key != "filter" && key != "columns" {
				return app, fmt.Errorf("unknown app.views[%d] field %q", i, key)
			}
		}
		view := AppView{}
		view.Title, ok = fields["title"].(string)
		if !ok || view.Title == "" || !appSingleLine(view.Title) || seen[view.Title] {
			return app, fmt.Errorf("app.views[%d] needs a unique, nonblank, single-line title", i)
		}
		seen[view.Title] = true
		if value, exists := fields["filter"]; exists {
			view.Filter, ok = value.(string)
			if !ok {
				return app, fmt.Errorf("app.views[%d].filter must be a string", i)
			}
		}
		if value, exists := fields["columns"]; exists {
			columns, ok := value.([]any)
			if !ok || len(columns) > 6 {
				return app, fmt.Errorf("app.views[%d].columns must be an array of at most six fields", i)
			}
			for _, value := range columns {
				column, ok := value.(string)
				if !ok || !appSingleLine(column) || (column != "id" && column != "title" && (!strings.HasPrefix(column, "data.") || len(column) == 5)) {
					return app, fmt.Errorf("app.views[%d]: columns must be id, title, or data.FIELD", i)
				}
				view.Columns = append(view.Columns, column)
			}
		}
		app.Views = append(app.Views, view)
	}
	return app, nil
}

func appSingleLine(value string) bool {
	return strings.Join(strings.Fields(value), " ") == value && strings.IndexFunc(value, unicode.IsControl) < 0
}
