/*
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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MultiClusterLister helps list MultiClusters.
// All objects returned here must be treated as read-only.
type MultiClusterLister interface {
	// List lists all MultiClusters in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.MultiCluster, err error)
	// MultiClusters returns an object that can list and get MultiClusters.
	MultiClusters(namespace string) MultiClusterNamespaceLister
	MultiClusterListerExpansion
}

// multiClusterLister implements the MultiClusterLister interface.
type multiClusterLister struct {
	indexer cache.Indexer
}

// NewMultiClusterLister returns a new MultiClusterLister.
func NewMultiClusterLister(indexer cache.Indexer) MultiClusterLister {
	return &multiClusterLister{indexer: indexer}
}

// List lists all MultiClusters in the indexer.
func (s *multiClusterLister) List(selector labels.Selector) (ret []*v1alpha1.MultiCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MultiCluster))
	})
	return ret, err
}

// MultiClusters returns an object that can list and get MultiClusters.
func (s *multiClusterLister) MultiClusters(namespace string) MultiClusterNamespaceLister {
	return multiClusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MultiClusterNamespaceLister helps list and get MultiClusters.
// All objects returned here must be treated as read-only.
type MultiClusterNamespaceLister interface {
	// List lists all MultiClusters in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.MultiCluster, err error)
	// Get retrieves the MultiCluster from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.MultiCluster, error)
	MultiClusterNamespaceListerExpansion
}

// multiClusterNamespaceLister implements the MultiClusterNamespaceLister
// interface.
type multiClusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MultiClusters in the indexer for a given namespace.
func (s multiClusterNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MultiCluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MultiCluster))
	})
	return ret, err
}

// Get retrieves the MultiCluster from the indexer for a given namespace and name.
func (s multiClusterNamespaceLister) Get(name string) (*v1alpha1.MultiCluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("multicluster"), name)
	}
	return obj.(*v1alpha1.MultiCluster), nil
}
