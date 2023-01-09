package utils

import (
	"bytes"
	"encoding/json"
)

func Equal(a interface{}, b interface{}) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}
