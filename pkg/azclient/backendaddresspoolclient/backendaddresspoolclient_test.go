// /*
// Copyright The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Code generated by client-gen. DO NOT EDIT.
package backendaddresspoolclient

import (
	"context"
	"strings"

	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var beforeAllFunc func(context.Context)
var afterAllFunc func(context.Context)
var additionalTestCases func()
var newResource *armnetwork.BackendAddressPool = &armnetwork.BackendAddressPool{}

var _ = ginkgo.Describe("LoadBalancerBackendAddressPoolsClient", ginkgo.Ordered, func() {

	if beforeAllFunc != nil {
		ginkgo.BeforeAll(beforeAllFunc)
	}

	if additionalTestCases != nil {
		additionalTestCases()
	}

	ginkgo.When("creation requests are raised", func() {
		ginkgo.It("should not return error", func(ctx context.Context) {
			newResource, err := realClient.CreateOrUpdate(ctx, resourceGroupName, loadbalancerName, resourceName, *newResource)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(newResource).NotTo(gomega.BeNil())
			gomega.Expect(strings.EqualFold(*newResource.Name, resourceName)).To(gomega.BeTrue())
		})
	})

	ginkgo.When("get requests are raised", func() {
		ginkgo.It("should not return error", func(ctx context.Context) {
			newResource, err := realClient.Get(ctx, resourceGroupName, loadbalancerName, resourceName)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(newResource).NotTo(gomega.BeNil())
		})
	})
	ginkgo.When("invalid get requests are raised", func() {
		ginkgo.It("should return 404 error", func(ctx context.Context) {
			newResource, err := realClient.Get(ctx, resourceGroupName, loadbalancerName, resourceName+"notfound")
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(newResource).To(gomega.BeNil())
		})
	})

	ginkgo.When("update requests are raised", func() {
		ginkgo.It("should not return error", func(ctx context.Context) {
			newResource, err := realClient.CreateOrUpdate(ctx, resourceGroupName, loadbalancerName, resourceName, *newResource)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(newResource).NotTo(gomega.BeNil())
		})
	})

	ginkgo.When("list requests are raised", func() {
		ginkgo.It("should not return error", func(ctx context.Context) {
			resourceList, err := realClient.List(ctx, resourceGroupName, loadbalancerName)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resourceList).NotTo(gomega.BeNil())
			gomega.Expect(len(resourceList)).To(gomega.Equal(1))
			gomega.Expect(*resourceList[0].Name).To(gomega.Equal(resourceName))
		})
	})
	ginkgo.When("invalid list requests are raised", func() {
		ginkgo.It("should return error", func(ctx context.Context) {
			resourceList, err := realClient.List(ctx, resourceGroupName+"notfound", loadbalancerName)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(resourceList).To(gomega.BeNil())
		})
	})

	ginkgo.When("deletion requests are raised", func() {
		ginkgo.It("should not return error", func(ctx context.Context) {
			err = realClient.Delete(ctx, resourceGroupName, loadbalancerName, resourceName)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	if afterAllFunc != nil {
		ginkgo.AfterAll(afterAllFunc)
	}
})