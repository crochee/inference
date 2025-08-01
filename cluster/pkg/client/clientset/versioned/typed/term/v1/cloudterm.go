/*
Copyright 2025.

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

package v1

import (
	termv1 "cluster/api/term/v1"
	applyconfigurationtermv1 "cluster/pkg/client/applyconfiguration/term/v1"
	scheme "cluster/pkg/client/clientset/versioned/scheme"
	context "context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// CloudTermsGetter has a method to return a CloudTermInterface.
// A group's client should implement this interface.
type CloudTermsGetter interface {
	CloudTerms(namespace string) CloudTermInterface
}

// CloudTermInterface has methods to work with CloudTerm resources.
type CloudTermInterface interface {
	Create(ctx context.Context, cloudTerm *termv1.CloudTerm, opts metav1.CreateOptions) (*termv1.CloudTerm, error)
	Update(ctx context.Context, cloudTerm *termv1.CloudTerm, opts metav1.UpdateOptions) (*termv1.CloudTerm, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, cloudTerm *termv1.CloudTerm, opts metav1.UpdateOptions) (*termv1.CloudTerm, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*termv1.CloudTerm, error)
	List(ctx context.Context, opts metav1.ListOptions) (*termv1.CloudTermList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *termv1.CloudTerm, err error)
	Apply(ctx context.Context, cloudTerm *applyconfigurationtermv1.CloudTermApplyConfiguration, opts metav1.ApplyOptions) (result *termv1.CloudTerm, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, cloudTerm *applyconfigurationtermv1.CloudTermApplyConfiguration, opts metav1.ApplyOptions) (result *termv1.CloudTerm, err error)
	CloudTermExpansion
}

// cloudTerms implements CloudTermInterface
type cloudTerms struct {
	*gentype.ClientWithListAndApply[*termv1.CloudTerm, *termv1.CloudTermList, *applyconfigurationtermv1.CloudTermApplyConfiguration]
}

// newCloudTerms returns a CloudTerms
func newCloudTerms(c *TermV1Client, namespace string) *cloudTerms {
	return &cloudTerms{
		gentype.NewClientWithListAndApply[*termv1.CloudTerm, *termv1.CloudTermList, *applyconfigurationtermv1.CloudTermApplyConfiguration](
			"cloudterms",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *termv1.CloudTerm { return &termv1.CloudTerm{} },
			func() *termv1.CloudTermList { return &termv1.CloudTermList{} },
		),
	}
}
