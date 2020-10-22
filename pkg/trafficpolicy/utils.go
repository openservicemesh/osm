package trafficpolicy

import "reflect"

// Equal returns true if the routes' fields are equal
func (r HTTPRoute) Equal(route HTTPRoute) bool {
	return reflect.DeepEqual(r, route)
}
