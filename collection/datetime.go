package collection

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// DateTimeFilter compares a string field directly under frontmatter.data.
// Comparisons use instants, so different timezone offsets can denote equality.
// Invalid filter inputs fail the operation. A document with a missing, non-string,
// or invalid timestamp does not match. All supplied filters must match.
type DateTimeFilter struct {
	// Field is a nonempty literal key under data. Dots are part of the key;
	// they do not select nested fields.
	Field string `json:"field"`
	// Op is "before", "after", "on_or_before", or "on_or_after". The first
	// two are exclusive; the on_or variants include equality.
	Op string `json:"op"`
	// Value is an RFC 3339 timestamp with a timezone and at most nine fractional
	// digits, for example "2026-11-03T09:00:00+01:00". Date-only values and leap
	// seconds are invalid. No schema annotation is required for the field.
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
