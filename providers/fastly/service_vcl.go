// Copyright 2019 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fastly

import (
	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/fastly/go-fastly/v6/fastly"
)

type ServiceVCLGenerator struct {
	FastlyService
}

func (g *ServiceVCLGenerator) loadServices(client *fastly.Client, filters []terraformutils.ResourceFilter) ([]*fastly.Service, error) {

	serviceIDs := []string{}
	for _, filter := range g.Filter {
		if filter.FieldPath == "id" {
			serviceIDs = append(serviceIDs, filter.AcceptableValues...)
		}
	}

	var err error
	var services []*fastly.Service
	if len(serviceIDs) > 0 {
		services = make([]*fastly.Service, len(serviceIDs))
		for i, serviceID := range serviceIDs {
			services[i], err = client.GetService(&fastly.GetServiceInput{serviceID})
			if err != nil {
				return nil, err
			}
		}
	} else {
		services, err = client.ListServices(&fastly.ListServicesInput{})
		if err != nil {
			return nil, err
		}
	}

	for _, service := range services {
		if service.Type == ServiceTypeVCL {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				service.ID,
				service.ID,
				"fastly_service_vcl",
				"fastly",
				[]string{}))
		} else if service.Type == ServiceTypeWasm {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				service.ID,
				service.ID,
				"fastly_service_compute",
				"fastly",
				[]string{}))
		}
	}
	return services, nil
}

func (g *ServiceVCLGenerator) loadDictionaryItems(client *fastly.Client, serviceID string) error {
	latest, err := client.LatestVersion(&fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	dictionaries, err := client.ListDictionaries(&fastly.ListDictionariesInput{
		ServiceID:      serviceID,
		ServiceVersion: latest.Number,
	})
	if err != nil {
		return err
	}
	for _, dictionary := range dictionaries {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			dictionary.ID,
			dictionary.ID,
			"fastly_service_dictionary_items",
			"fastly",
			map[string]string{
				"service_id":    serviceID,
				"dictionary_id": dictionary.ID,
			},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *ServiceVCLGenerator) loadACLEntries(client *fastly.Client, serviceID string) error {
	latest, err := client.LatestVersion(&fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	acls, err := client.ListACLs(&fastly.ListACLsInput{
		ServiceID:      serviceID,
		ServiceVersion: latest.Number,
	})
	if err != nil {
		return err
	}
	for _, acl := range acls {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			acl.ID,
			acl.ID,
			"fastly_service_acl_entries",
			"fastly",
			map[string]string{
				"service_id": serviceID,
				"acl_id":     acl.ID,
			},
			[]string{},
			map[string]interface{}{}))
	}
	return nil
}

func (g *ServiceVCLGenerator) loadDynamicSnippetContent(client *fastly.Client, serviceID string) error {
	latest, err := client.LatestVersion(&fastly.LatestVersionInput{
		ServiceID: serviceID,
	})
	if err != nil {
		return err
	}
	snippets, err := client.ListSnippets(&fastly.ListSnippetsInput{
		ServiceID:      serviceID,
		ServiceVersion: latest.Number,
	})
	if err != nil {
		return err
	}
	for _, snippet := range snippets {
		// check if dynamic
		if snippet.Dynamic == 1 {
			g.Resources = append(g.Resources, terraformutils.NewResource(
				snippet.ID,
				snippet.ID,
				"fastly_service_dynamic_snippet_content",
				"fastly",
				map[string]string{
					"service_id": serviceID,
					"snippet_id": snippet.ID,
				},
				[]string{},
				map[string]interface{}{}))
		}
	}
	return nil
}

func (g *ServiceVCLGenerator) InitResources() error {
	client, err := fastly.NewClient(g.Args["api_key"].(string))
	if err != nil {
		return err
	}

	services, err := g.loadServices(client, g.Filter)
	if err != nil {
		return err
	}
	_ = services
	for _, service := range services {
		err := g.loadDictionaryItems(client, service.ID)
		if err != nil {
			return err
		}
		err = g.loadACLEntries(client, service.ID)
		if err != nil {
			return err
		}
		err = g.loadDynamicSnippetContent(client, service.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
