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

package v1

type NodeInfo struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	Owner            string `json:"owner"`
	WorkloadName     string `json:"workload_name"`
	IstioVersion     string `json:"istio_version"`
	MeshId           string `json:"mesh_id"`
	ClusterId        string `json:"cluster_idd"`
	Labels           string `json:"labels"`
	PlatformMetadata string `json:"platform_metadata"`
	AppContainers    string `json:"app_containers"`
	InstanceIps      string `json:"instance_ips"`
}
