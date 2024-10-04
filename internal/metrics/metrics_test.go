package metrics

import (
	"fmt"
	"testing"
)

func TestCanPassNilTags(t *testing.T) {
	logEvent := func(_ string, tags map[string]string, fields map[string]interface{}) {
		for key, value := range tags {
			fmt.Println(key, value)
		}

		for key, value := range fields {
			fmt.Println(key, value)
		}
	}

	t.Run("Empty tags and field", func(_ *testing.T) {
		logEvent("test", nil, nil)
	})
}
