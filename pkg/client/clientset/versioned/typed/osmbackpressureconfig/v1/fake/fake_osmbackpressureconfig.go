/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	osmbackpressureconfigv1 "github.com/open-service-mesh/osm/pkg/apis/osmbackpressureconfig/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeOSMBackpressureConfigs implements OSMBackpressureConfigInterface
type FakeOSMBackpressureConfigs struct {
	Fake *FakeOsmbackpressureconfigV1
	ns   string
}

var osmbackpressureconfigsResource = schema.GroupVersionResource{Group: "osmbackpressureconfig.openservicemesh.io", Version: "v1", Resource: "osmbackpressureconfigs"}

var osmbackpressureconfigsKind = schema.GroupVersionKind{Group: "osmbackpressureconfig.openservicemesh.io", Version: "v1", Kind: "OSMBackpressureConfig"}

// Get takes name of the oSMBackpressureConfig, and returns the corresponding oSMBackpressureConfig object, and an error if there is any.
func (c *FakeOSMBackpressureConfigs) Get(ctx context.Context, name string, options v1.GetOptions) (result *osmbackpressureconfigv1.OSMBackpressureConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(osmbackpressureconfigsResource, c.ns, name), &osmbackpressureconfigv1.OSMBackpressureConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*osmbackpressureconfigv1.OSMBackpressureConfig), err
}

// List takes label and field selectors, and returns the list of OSMBackpressureConfigs that match those selectors.
func (c *FakeOSMBackpressureConfigs) List(ctx context.Context, opts v1.ListOptions) (result *osmbackpressureconfigv1.OSMBackpressureConfigList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(osmbackpressureconfigsResource, osmbackpressureconfigsKind, c.ns, opts), &osmbackpressureconfigv1.OSMBackpressureConfigList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &osmbackpressureconfigv1.OSMBackpressureConfigList{ListMeta: obj.(*osmbackpressureconfigv1.OSMBackpressureConfigList).ListMeta}
	for _, item := range obj.(*osmbackpressureconfigv1.OSMBackpressureConfigList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested oSMBackpressureConfigs.
func (c *FakeOSMBackpressureConfigs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(osmbackpressureconfigsResource, c.ns, opts))

}

// Create takes the representation of a oSMBackpressureConfig and creates it.  Returns the server's representation of the oSMBackpressureConfig, and an error, if there is any.
func (c *FakeOSMBackpressureConfigs) Create(ctx context.Context, oSMBackpressureConfig *osmbackpressureconfigv1.OSMBackpressureConfig, opts v1.CreateOptions) (result *osmbackpressureconfigv1.OSMBackpressureConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(osmbackpressureconfigsResource, c.ns, oSMBackpressureConfig), &osmbackpressureconfigv1.OSMBackpressureConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*osmbackpressureconfigv1.OSMBackpressureConfig), err
}

// Update takes the representation of a oSMBackpressureConfig and updates it. Returns the server's representation of the oSMBackpressureConfig, and an error, if there is any.
func (c *FakeOSMBackpressureConfigs) Update(ctx context.Context, oSMBackpressureConfig *osmbackpressureconfigv1.OSMBackpressureConfig, opts v1.UpdateOptions) (result *osmbackpressureconfigv1.OSMBackpressureConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(osmbackpressureconfigsResource, c.ns, oSMBackpressureConfig), &osmbackpressureconfigv1.OSMBackpressureConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*osmbackpressureconfigv1.OSMBackpressureConfig), err
}

// Delete takes name of the oSMBackpressureConfig and deletes it. Returns an error if one occurs.
func (c *FakeOSMBackpressureConfigs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(osmbackpressureconfigsResource, c.ns, name), &osmbackpressureconfigv1.OSMBackpressureConfig{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOSMBackpressureConfigs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(osmbackpressureconfigsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &osmbackpressureconfigv1.OSMBackpressureConfigList{})
	return err
}

// Patch applies the patch and returns the patched oSMBackpressureConfig.
func (c *FakeOSMBackpressureConfigs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *osmbackpressureconfigv1.OSMBackpressureConfig, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(osmbackpressureconfigsResource, c.ns, name, pt, data, subresources...), &osmbackpressureconfigv1.OSMBackpressureConfig{})

	if obj == nil {
		return nil, err
	}
	return obj.(*osmbackpressureconfigv1.OSMBackpressureConfig), err
}
