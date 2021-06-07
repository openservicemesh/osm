package utils

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	yaml "gopkg.in/yaml.v2"
)

// ProtoToYAML converts a Proto message to it's YAML representation in bytes
func ProtoToYAML(m protoreflect.ProtoMessage) ([]byte, error) {
	marshalOptions := protojson.MarshalOptions{
		UseProtoNames: true,
	}
	configJSON, err := marshalOptions.Marshal(m)
	if err != nil {
		return nil, err
	}

	configYAML, err := jsonToYAML(configJSON)
	if err != nil {
		log.Error().Err(err).Msgf("Error marshaling xDS struct into YAML")
		return nil, err
	}
	return configYAML, err
}

// jsonToYAML converts a JSON representation in bytes to the corresponding YAML representation in bytes
// Reference impl taken from https://github.com/ghodss/yaml/blob/master/yaml.go#L87
func jsonToYAML(jb []byte) ([]byte, error) {
	// Convert the JSON to an object.
	var jsonObj interface{}
	// We are using yaml.Unmarshal here (instead of json.Unmarshal) because the
	// Go JSON library doesn't try to pick the right number type (int, float,
	// etc.) when unmarshalling to interface{}, it just picks float64
	// universally. go-yaml does go through the effort of picking the right
	// number type, so we can preserve number type throughout this process.
	err := yaml.Unmarshal([]byte(jb), &jsonObj)
	if err != nil {
		return nil, err
	}

	// Marshal this object into YAML.
	return yaml.Marshal(jsonObj)
}
