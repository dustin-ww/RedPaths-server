package dgraph

import (
	"encoding/json"
	"fmt"
)

func UnmarshalResponse(data []byte, target any) error {
	if len(data) == 0 {
		return fmt.Errorf("empty response")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal dgraph response: %w", err)
	}
	return nil
}
