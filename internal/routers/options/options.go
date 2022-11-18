// Copyright The CSM Authors.
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

package options

import (
	// versionedclient "istio.io/client-go/pkg/clientset/versioned"
	// istioinformer "istio.io/client-go/pkg/informers/externalversions"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Options for lazysidecar
type Options struct {
	// IstioClient   *versionedclient.Clientset
	// IstioInformer istioinformer.SharedInformerFactory
	Manager *ctrl.Manager
}

// New creates an Options
// func New(ic *versionedclient.Clientset) *Options {
func New(mgr *ctrl.Manager) *Options {
	return &Options{
		// IstioClient:   ic,
		// IstioInformer: istioinformer.NewSharedInformerFactory(ic, 0),
		Manager: mgr,
	}
}
