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

package utils

import (
	"fmt"

	istioclient "istio.io/client-go/pkg/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewKubeClient creates new kube client
func NewKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	var err error
	var kubeConf *rest.Config

	if kubeconfigPath == "" {
		// creates the in-cluster config
		kubeConf = ctrl.GetConfigOrDie()
		// if err != nil {
		// 	return nil, fmt.Errorf("build default in cluster kube config failed: %w", err)
		// }
	} else {
		kubeConf, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("build kube client config from config file failed: %w", err)
		}
	}
	return kubernetes.NewForConfig(kubeConf)
}

// NewIstioClient creates new istio client which use to handle istio CRD
func NewIstioClient(kubeconfigPath string) (*istioclient.Clientset, error) {
	var err error
	var istioConf *rest.Config

	if kubeconfigPath == "" {
		istioConf = ctrl.GetConfigOrDie()
		// istioConf, err = rest.InClusterConfig()
		// if err != nil {
		// 	return nil, fmt.Errorf("build default in cluster istio config failed: %w", err)
		// }
	} else {
		istioConf, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("build istio client config from config file failed: %w", err)
		}
	}

	return istioclient.NewForConfig(istioConf)
}
