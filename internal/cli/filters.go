package cli

import (
	"fmt"
	"github.com/hmans/means/internal/collection"
	"github.com/spf13/cobra"
	"strings"
)

type filterFlags struct {
	exact    []string
	datetime map[string]*[]string
}

func (f *filterFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringArrayVar(&f.exact, "filter", nil, "Match a data field as KEY=VALUE (repeatable; values may be JSON)")
	f.datetime = map[string]*[]string{}
	for _, flag := range []string{"before", "after", "on-or-before", "on-or-after"} {
		f.datetime[flag] = cmd.Flags().StringArray(flag, nil, "Compare a data datetime as KEY=TIMESTAMP (RFC 3339; repeatable)")
	}
}

func (f *filterFlags) parse() (map[string]any, []collection.DateTimeFilter, error) {
	exact, err := parseSearchFilters(f.exact)
	if err != nil {
		return nil, nil, err
	}
	var datetime []collection.DateTimeFilter
	for _, flag := range []string{"before", "after", "on-or-before", "on-or-after"} {
		for _, raw := range *f.datetime[flag] {
			field, value, ok := strings.Cut(raw, "=")
			if !ok || field == "" {
				return nil, nil, fmt.Errorf("--%s must be KEY=TIMESTAMP with a nonempty data field name", flag)
			}
			datetime = append(datetime, collection.DateTimeFilter{Field: field, Op: strings.ReplaceAll(flag, "-", "_"), Value: value})
		}
	}
	return exact, datetime, nil
}
