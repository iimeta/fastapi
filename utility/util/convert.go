package util

import (
	"encoding/json"

	"github.com/gogf/gf/v2/util/gconv"
)

func ConvToMap(value any) map[string]any {

	if value == nil {
		return nil
	}

	if v, ok := value.([]byte); ok {
		data := make(map[string]any)
		if err := json.Unmarshal(v, &data); err == nil {
			return data
		}
	}

	return gconv.Map(value)
}
