// Copyright 2024 Google LLC
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

package registry

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/config"
	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/controller/direct/directbase"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var singleton = registry{}

type registry struct {
	registrations map[schema.GroupKind]*registration
}

type registration struct {
	gvk     schema.GroupVersionKind
	factory ModelFactoryFunc
	model   directbase.Model
}

type ModelFactoryFunc func(ctx context.Context, config *config.ControllerConfig) (directbase.Model, error)

func GetModel(gk schema.GroupKind) (directbase.Model, error) {
	registration := singleton.registrations[gk]
	if registration == nil || registration.model == nil {
		return nil, fmt.Errorf("no model registered for %s", gk)
	}
	return registration.model, nil
}

func PreferredGVK(gk schema.GroupKind) (schema.GroupVersionKind, bool) {
	registration := singleton.registrations[gk]
	if registration == nil {
		return schema.GroupVersionKind{}, false
	}
	return registration.gvk, true
}

func Init(ctx context.Context, config *config.ControllerConfig) error {
	for _, registration := range singleton.registrations {
		model, err := registration.factory(ctx, config)
		if err != nil {
			return err
		}

		registration.model = model
	}
	return nil
}

func RegisterModel(gvk schema.GroupVersionKind, modelFn ModelFactoryFunc) {
	if singleton.registrations == nil {
		singleton.registrations = make(map[schema.GroupKind]*registration)
	}
	singleton.registrations[gvk.GroupKind()] = &registration{
		gvk:     gvk,
		factory: modelFn,
	}
}

func IsDirectByGK(gk schema.GroupKind) bool {
	registration := singleton.registrations[gk]
	return registration != nil
}

// IsIAMDirect returns true if this resource uses the direct-reconciliation model for IAM.
func IsIAMDirect(groupKind schema.GroupKind) bool {
	registration := singleton.registrations[groupKind]
	if registration == nil {
		return false
	}

	// TODO: Move to registration somehow?
	switch groupKind {
	case schema.GroupKind{Group: "privateca.cnrm.cloud.google.com", Kind: "PrivateCACAPool"}:
		return true
	}
	return false
}

// SupportsIAM returns true if this resource supports IAM (not all GCP resources do).
// An error will be returned if IsDirect(groupKind) is not true.
func SupportsIAM(groupKind schema.GroupKind) (bool, error) {
	// TODO: Move to registration somehow?
	switch groupKind {
	case schema.GroupKind{Group: "logging.cnrm.cloud.google.com", Kind: "LoggingLogMetric"}:
		return false, nil
	case schema.GroupKind{Group: "monitoring.cnrm.cloud.google.com", Kind: "MonitoringDashboard"}:
		return false, nil
	}
	return false, fmt.Errorf("groupKind %v is not recognized as a direct kind", groupKind)
}
