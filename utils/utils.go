package utils

import "encoding/json"

func Marshal(i interface{}) string {
	bytes, _ := json.Marshal(i)
	return string(bytes)
}
