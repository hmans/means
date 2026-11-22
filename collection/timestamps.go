package collection

import (
	"fmt"
	"time"
)

// Means maintains these frontmatter fields: created_at is set once on creation,
// updated_at on every write that changes a document. Values are RFC 3339 UTC
// timestamps with second precision. Documents written before timestamps were
// introduced may lack them; means never invents a creation time.
const (
	CreatedAtField = "created_at"
	UpdatedAtField = "updated_at"
)

// clock is replaced in tests.
var clock = time.Now

func timestamp() string {
	return clock().UTC().Format(time.RFC3339)
}

func isTimestampField(key string) bool {
	return key == CreatedAtField || key == UpdatedAtField
}

func validateTimestamps(frontmatter map[string]any) error {
	for _, key := range []string{CreatedAtField, UpdatedAtField} {
		value, exists := frontmatter[key]
		if !exists {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("frontmatter.%s must be an RFC 3339 timestamp string", key)
		}
		if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
			return fmt.Errorf("frontmatter.%s must be an RFC 3339 timestamp with a timezone offset: %q", key, text)
		}
	}
	return nil
}

func rejectTimestamps(frontmatter map[string]any) error {
	for key := range frontmatter {
		if isTimestampField(key) {
			return fmt.Errorf("%s is maintained by means; omit it", key)
		}
	}
	return nil
}
