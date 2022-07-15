package utils

import (
	"encoding/json"
	"fmt"
)

// PrettyJSON Unmarshals and Marshall again with Indent so it is human readable
func PrettyJSON(js []byte, prefix string) ([]byte, error) {
	var jsonObj interface{}
	err := json.Unmarshal(js, &jsonObj)
	if err != nil {
		return nil, fmt.Errorf("Could not Unmarshal a byte array: %w", err)
	}
	return json.MarshalIndent(jsonObj, prefix, "    ")
}
