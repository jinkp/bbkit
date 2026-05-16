package output

import (
	"encoding/json"
)

func PrintJSON(v any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
