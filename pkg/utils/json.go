package utils

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// PrettyJSON Unmarshals and Marshall again with Indent so it is human readable
func PrettyJSON(js []byte, prefix string) ([]byte, error) {
	var jsonObj interface{}
	err := json.Unmarshal(js, &jsonObj)
	if err != nil {
		return nil, errors.Wrap(err, "Could not Unmarshal a byte array")
	}
	return json.MarshalIndent(jsonObj, prefix, "    ")
}
