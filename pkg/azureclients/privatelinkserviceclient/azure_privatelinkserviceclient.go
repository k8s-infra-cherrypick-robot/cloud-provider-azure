/*
Copyright 2021 The Kubernetes Authors.

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

// Package privatelinkserviceclient implements the client for PrivateLinkService.
package privatelinkserviceclient

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/klog/v2"

	azclients "sigs.k8s.io/cloud-provider-azure/pkg/azureclients"
	"sigs.k8s.io/cloud-provider-azure/pkg/azureclients/armclient"
	"sigs.k8s.io/cloud-provider-azure/pkg/metrics"
	"sigs.k8s.io/cloud-provider-azure/pkg/retry"
)

var _ Interface = &Client{}

const (
	PLSResourceKinkd = "Microsoft.Network/privatelinkservices"
)

// Client implements privatelinkservice Interface.
type Client struct {
	armClient      armclient.Interface
	cloudName      string
	subscriptionID string

	// Rate limiting configures.
	rateLimiterReader flowcontrol.RateLimiter
	rateLimiterWriter flowcontrol.RateLimiter

	// ARM throttling configures.
	RetryAfterReader time.Time
	RetryAfterWriter time.Time
}

// New creates a new private link service client.
func New(config *azclients.ClientConfig) *Client {

	apiVersion := APIVersion
	if strings.EqualFold(config.CloudName, AzureStackCloudName) && !config.DisableAzureStackCloud {
		apiVersion = AzureStackCloudAPIVersion
	}
	armClient := armclient.New(config.Authorizer, *config, config.ResourceManagerEndpoint, apiVersion)

	rateLimiterReader, rateLimiterWriter := azclients.NewRateLimiter(config.RateLimitConfig)
	if azclients.RateLimitEnabled(config.RateLimitConfig) {
		klog.V(2).Infof("Azure PrivateLinkServicesClient (read ops) using rate limit config: QPS=%g, bucket=%d",
			config.RateLimitConfig.CloudProviderRateLimitQPS,
			config.RateLimitConfig.CloudProviderRateLimitBucket)
		klog.V(2).Infof("Azure PrivateLinkServicesClient (write ops) using rate limit config: QPS=%g, bucket=%d",
			config.RateLimitConfig.CloudProviderRateLimitQPSWrite,
			config.RateLimitConfig.CloudProviderRateLimitBucketWrite)
	}

	client := &Client{
		armClient:         armClient,
		rateLimiterReader: rateLimiterReader,
		rateLimiterWriter: rateLimiterWriter,
		subscriptionID:    config.SubscriptionID,
		cloudName:         config.CloudName,
	}
	return client
}

// CreateOrUpdate creates or updates a private link service .
func (c *Client) CreateOrUpdate(ctx context.Context, resourceGroupName string, privateLinkServiceName string, privateLinkService network.PrivateLinkService, etag string) *retry.Error {
	mc := metrics.NewMetricContext("private_link_services", "create_or_update", resourceGroupName, c.subscriptionID, "")

	// Report errors if the client is rate limited.
	if !c.rateLimiterWriter.TryAccept() {
		mc.RateLimitedCount()
		return retry.GetRateLimitError(true, "PLSCreateOrUpdate")
	}

	// Report errors if the client is throttled.
	if c.RetryAfterWriter.After(time.Now()) {
		mc.ThrottledCount()
		rerr := retry.GetThrottlingError("PLSCreateOrUpdate", "client throttled", c.RetryAfterWriter)
		return rerr
	}

	rerr := c.createOrUpdatePLS(ctx, resourceGroupName, privateLinkServiceName, privateLinkService, etag)
	mc.Observe(rerr)
	if rerr != nil {
		if rerr.IsThrottled() {
			// Update RetryAfterReader so that no more requests would be sent until RetryAfter expires.
			c.RetryAfterWriter = rerr.RetryAfter
		}

		return rerr
	}
	return nil
}
func (c *Client) createOrUpdatePLS(ctx context.Context, resourceGroupName string, privateLinkServiceName string, parameters network.PrivateLinkService, etag string) *retry.Error {
	resourceID := armclient.GetResourceID(
		c.subscriptionID,
		resourceGroupName,
		PLSResourceKinkd,
		privateLinkServiceName,
	)
	decorators := []autorest.PrepareDecorator{
		autorest.WithPathParameters("{resourceID}", map[string]interface{}{"resourceID": resourceID}),
		autorest.WithJSON(parameters),
	}
	if etag != "" {
		decorators = append(decorators, autorest.WithHeader("If-Match", autorest.String(etag)))
	}

	response, rerr := c.armClient.PutResourceWithDecorators(ctx, resourceID, parameters, decorators)
	defer c.armClient.CloseResponse(ctx, response)
	if rerr != nil {
		klog.V(5).Infof("Received error in %s: resourceID: %s, error: %s", "privatelinkservice.put.request", resourceID, rerr.Error())
		return rerr
	}

	if response != nil && response.StatusCode != http.StatusNoContent {
		_, rerr = c.createOrUpdateResponder(response)
		if rerr != nil {
			klog.V(5).Infof("Received error in %s: resourceID: %s, error: %s", "privatelinkservice.put.respond", resourceID, rerr.Error())
			return rerr
		}
	}
	return nil
}

func (c *Client) createOrUpdateResponder(resp *http.Response) (*network.PrivateLinkService, *retry.Error) {
	result := &network.PrivateLinkService{}
	err := autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result))
	result.Response = autorest.Response{Response: resp}
	return result, retry.GetError(resp, err)
}

// Get gets the private link service
func (c *Client) Get(ctx context.Context, resourceGroupName string, privateLinkServiceName string, expand string) (network.PrivateLinkService, *retry.Error) {
	mc := metrics.NewMetricContext("private_link_services", "get", resourceGroupName, c.subscriptionID, "")

	// Report errors if the client is rate limited.
	if !c.rateLimiterReader.TryAccept() {
		mc.RateLimitedCount()
		return network.PrivateLinkService{}, retry.GetRateLimitError(false, "PLSGet")
	}

	// Report errors if the client is throttled.
	if c.RetryAfterReader.After(time.Now()) {
		mc.ThrottledCount()
		rerr := retry.GetThrottlingError("PLSGet", "client throttled", c.RetryAfterReader)
		return network.PrivateLinkService{}, rerr
	}
	result, rerr := c.getPLS(ctx, resourceGroupName, privateLinkServiceName, expand)

	mc.Observe(rerr)
	if rerr != nil {
		if rerr.IsThrottled() {
			// Update RetryAfterReader so that no more requests would be sent until RetryAfter expires.
			c.RetryAfterReader = rerr.RetryAfter
		}
		return result, rerr
	}
	return result, nil
}

// getPLS gets a privatelinkservice.
func (c *Client) getPLS(ctx context.Context, resourceGroupName string, privateLinkServiceName string, expand string) (network.PrivateLinkService, *retry.Error) {
	resourceID := armclient.GetResourceID(
		c.subscriptionID,
		resourceGroupName,
		PLSResourceKinkd,
		privateLinkServiceName,
	)
	result := network.PrivateLinkService{}
	response, rerr := c.armClient.GetResource(ctx, resourceID, expand)
	defer c.armClient.CloseResponse(ctx, response)
	if rerr != nil {
		klog.V(5).Infof("Received error in %s: resourceID: %s, error: %s", "privatelinkservice.get.request", resourceID, rerr.Error())
		return result, rerr
	}

	err := autorest.Respond(
		response,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result))
	if err != nil {
		klog.V(5).Infof("Received error in %s: resourceID: %s, error: %s", "privatelinkservice.get.respond", resourceID, err)
		return result, retry.GetError(response, err)
	}

	result.Response = autorest.Response{Response: response}
	return result, nil
}

func (c *Client) Delete(ctx context.Context, resourceGroupName string, privateLinkServiceName string) *retry.Error {
	mc := metrics.NewMetricContext("private_link_services", "delete", resourceGroupName, c.subscriptionID, "")

	// Report errors if the client is rate limited.
	if !c.rateLimiterWriter.TryAccept() {
		mc.RateLimitedCount()
		return retry.GetRateLimitError(true, "PLSDelete")
	}

	// Report errors if the client is throttled.
	if c.RetryAfterWriter.After(time.Now()) {
		mc.ThrottledCount()
		rerr := retry.GetThrottlingError("PLSDelete", "client throttled", c.RetryAfterWriter)
		return rerr
	}

	rerr := c.deletePLS(ctx, resourceGroupName, privateLinkServiceName)
	mc.Observe(rerr)
	if rerr != nil {
		if rerr.IsThrottled() {
			// Update RetryAfterReader so that no more requests would be sent until RetryAfter expires.
			c.RetryAfterWriter = rerr.RetryAfter
		}

		return rerr
	}

	return nil
}

// deletePLS deletes a privatelinkservice by name.
func (c *Client) deletePLS(ctx context.Context, resourceGroupName string, privateLinkServiceName string) *retry.Error {
	resourceID := armclient.GetResourceID(
		c.subscriptionID,
		resourceGroupName,
		PLSResourceKinkd,
		privateLinkServiceName,
	)

	return c.armClient.DeleteResource(ctx, resourceID, "")
}
