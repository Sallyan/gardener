// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package shared

import (
	fluentbitv1alpha2 "github.com/fluent/fluent-operator/v2/apis/fluentbit/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/component"
	"github.com/gardener/gardener/pkg/component/logging/fluentoperator"
)

// NewFluentOperatorCustomResources instantiates a new `Fluent Operator Custom Resources` component.
func NewFluentOperatorCustomResources(
	c client.Client,
	gardenNamespaceName string,
	enabled bool,
	suffix string,
	centralLoggingConfigurations []component.CentralLoggingConfiguration,
	output *fluentbitv1alpha2.ClusterOutput,
) (
	deployer component.DeployWaiter,
	err error,
) {
	var (
		inputs  []*fluentbitv1alpha2.ClusterInput
		filters []*fluentbitv1alpha2.ClusterFilter
		parsers []*fluentbitv1alpha2.ClusterParser
	)

	// Fetch component specific logging configurations
	for _, componentFn := range centralLoggingConfigurations {
		loggingConfig, err := componentFn()
		if err != nil {
			return nil, err
		}

		if len(loggingConfig.Inputs) > 0 {
			inputs = append(inputs, loggingConfig.Inputs...)
		}

		if len(loggingConfig.Filters) > 0 {
			filters = append(filters, loggingConfig.Filters...)
		}

		if len(loggingConfig.Parsers) > 0 {
			parsers = append(parsers, loggingConfig.Parsers...)
		}
	}

	deployer = fluentoperator.NewCustomResources(
		c,
		gardenNamespaceName,
		fluentoperator.CustomResourcesValues{
			Suffix:  suffix,
			Inputs:  inputs,
			Filters: filters,
			Parsers: parsers,
			Outputs: []*fluentbitv1alpha2.ClusterOutput{output},
		},
	)

	if !enabled {
		deployer = component.OpDestroyAndWait(deployer)
	}

	return deployer, nil
}
