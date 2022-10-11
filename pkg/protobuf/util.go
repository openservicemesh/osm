// Package protobuf contains function(s) pertaining to protobufs
package protobuf

import (
	"fmt"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"google.golang.org/protobuf/types/known/anypb"
)

// MustMarshalAny marshals a protobuf Message into an Any type. It panics if that operation fails.
func MustMarshalAny(pb proto.Message) *any.Any {
	msg, err := anypb.New(proto.MessageV2(pb))
	if err != nil {
		panic(err.Error())
	}

	return msg
}

// ToJSON marshals a protobuf Message to a JSON string representation
func ToJSON(pb proto.Message) (string, error) {
	if pb == nil {
		return "", fmt.Errorf("unexpected nil proto.Message")
	}

	m := jsonpb.Marshaler{}
	return m.MarshalToString(pb)
}

// MustToJSON marshals a protobuf Message to a JSON string representation and panics
// if the marshalling fails
func MustToJSON(pb proto.Message) string {
	str, err := ToJSON(pb)
	if err != nil {
		panic(err.Error())
	}

	return str
}
