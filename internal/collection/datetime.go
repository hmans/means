package collection

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DateTimeFilter compares a string field directly under frontmatter.data.
// Explicit operators leave ordinary exact-value filters unchanged.
type DateTimeFilter struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type compiledDateTimeFilter struct {
	DateTimeFilter
	instant time.Time
}

// Bound precision explicitly: time.Parse otherwise silently truncates fractions.
var dateTimePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[Tt]\d{2}:\d{2}:\d{2}(\.[0-9]{1,9})?([Zz]|[+-]\d{2}:\d{2})$`)

func parseDateTime(value string) (time.Time, error) {
	if !dateTimePattern.MatchString(value) {
		return time.Time{}, fmt.Errorf("expected an RFC 3339 timestamp with timezone and at most 9 fractional digits")
	}
	// Go's parser permits out-of-range zone components; reject those explicitly.
	if value[len(value)-1] != 'Z' && value[len(value)-1] != 'z' {
		zone := value[len(value)-5:]
		if zone[:2] > "23" || zone[3:] > "59" {
			return time.Time{}, fmt.Errorf("invalid timezone offset")
		}
	}
	return time.Parse(time.RFC3339Nano, strings.ToUpper(value))
}

func compileDateTimeFilters(filters []DateTimeFilter) ([]compiledDateTimeFilter, error) {
	result := make([]compiledDateTimeFilter, 0, len(filters))
	for _, filter := range filters {
		if filter.Field == "" {
			return nil, fmt.Errorf("datetime filter field must not be empty")
		}
		switch filter.Op {
		case "before", "after", "on_or_before", "on_or_after":
		default:
			return nil, fmt.Errorf("unknown datetime filter operator %q", filter.Op)
		}
		instant, err := parseDateTime(filter.Value)
		if err != nil {
			return nil, fmt.Errorf("datetime filter %q: %w", filter.Field, err)
		}
		result = append(result, compiledDateTimeFilter{filter, instant})
	}
	return result, nil
}

func (opts SearchOptions) matchesData(data map[string]any) bool {
	if !MatchesFilter(data, opts.Filter) {
		return false
	}
	for _, filter := range opts.dateTimeFilters {
		value, ok := data[filter.Field].(string)
		if !ok {
			return false
		}
		instant, err := parseDateTime(value)
		if err != nil {
			return false
		}
		comparison := instant.Compare(filter.instant)
		switch filter.Op {
		case "before":
			if comparison >= 0 {
				return false
			}
		case "after":
			if comparison <= 0 {
				return false
			}
		case "on_or_before":
			if comparison > 0 {
				return false
			}
		case "on_or_after":
			if comparison < 0 {
				return false
			}
		}
	}
	return true
}
