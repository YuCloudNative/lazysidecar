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

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/pkg/log"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/yucloudnative/lazysidecar/api/v1"
	"github.com/yucloudnative/lazysidecar/internal/routers/options"
	res "github.com/yucloudnative/lazysidecar/pkg/response"
	"github.com/yucloudnative/lazysidecar/pkg/utils"
)

const (
	nodeLabelsKey                         = "LABELS"
	nodeNamespaceKey                      = "NAMESPACE"
	nodeLabelsKeyPodTemplateHash          = "pod-template-hash"
	nodeLabelsKeySecurityTlsMode          = "security.istio.io/tlsMode"
	nodeLabelsKeyServiceCanonicalName     = "service.istio.io/canonical-name"
	nodeLabelsKeyServiceCanonicalRevision = "service.istio.io/canonical-revision"

	FQDNSuffix = ".svc.cluster.local"
)

type Invocation struct {
	opts        *options.Options
	excludeKeys []string
}

type InvocationHeader struct {
	Source      string `header:"src"`
	Destination string `header:"des"`
}

func NewInvocation(opts *options.Options) Invocation {
	excludeKeys := make([]string, 0)
	excludeKeys = append(excludeKeys, nodeLabelsKeyPodTemplateHash)
	excludeKeys = append(excludeKeys, nodeLabelsKeySecurityTlsMode)
	excludeKeys = append(excludeKeys, nodeLabelsKeyServiceCanonicalName)
	excludeKeys = append(excludeKeys, nodeLabelsKeyServiceCanonicalRevision)

	return Invocation{
		opts:        opts,
		excludeKeys: excludeKeys,
	}
}

func (i Invocation) Report(c *gin.Context) {
	response := res.NewResponse(c)

	// ?????????
	h := InvocationHeader{}
	if err := c.ShouldBindHeader(&h); err != nil {
		c.JSON(200, err)
	}

	// ?????? source metadata
	workloadSelector, sn, err := i.ParseWorkloadLabels(h.Source)
	if err != nil {
		fmt.Println("parse workload labels failed")
		return
	}

	// ?????? destination
	host, err := i.ParseHost(sn, h.Destination)
	if err != nil {
		fmt.Println("parse destination host failed")
		return
	}

	// informer := i.opts.IstioInformer.Networking().V1alpha3().EnvoyFilters()
	// ??? apiserver ?????????????????? list
	// if !cache.WaitForCacheSync(c.Done(), informer.Informer().HasSynced) {
	// 	runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
	// 	return
	// }

	lazySidecars := &v1.LazySidecarList{}
	mgr := *i.opts.Manager
	if err := mgr.GetClient().List(c, lazySidecars); err != nil {
		log.Error(err, "unable to fetch LazySidecar")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return
	}
	fmt.Println("lazysidecar list:", lazySidecars)
	for _, ls := range lazySidecars.Items {
		ws := ls.Spec.WorkloadSelector
		if isSourceMatched(workloadSelector, ws) {
			i.syncLazySidecar(c, ls, host)
		}
	}

	response.ToResponse(gin.H{})
	return
}

func (i Invocation) ParseWorkloadLabels(sourceMetadata string) (map[string]string, string, error) {
	rawBytes, err := base64.StdEncoding.DecodeString(sourceMetadata)
	if err != nil {
		panic(err)
	}

	metadata := &structpb.Struct{}
	err = proto.Unmarshal(rawBytes, metadata)
	if err != nil {
		fmt.Println(err)
	}

	mdFields := metadata.GetFields()
	if len(mdFields) == 0 ||
		mdFields[nodeLabelsKey] == nil ||
		mdFields[nodeLabelsKey].GetStructValue() == nil ||
		mdFields[nodeLabelsKey].GetStructValue().GetFields() == nil {
		fmt.Printf("Get node labels failed.")
		// TODO: ?????? Error
		return nil, "", nil
	}

	workloadSelector := make(map[string]string)
	nodeLabels := mdFields[nodeLabelsKey].GetStructValue().GetFields()
	for k, v := range nodeLabels {
		if utils.IsInStringSlice(k, i.excludeKeys) {
			continue
		}
		if len(v.GetStringValue()) != 0 {
			workloadSelector[k] = v.GetStringValue()
		}
	}
	fmt.Printf("%#v\n", workloadSelector)

	sn := mdFields[nodeNamespaceKey].GetStringValue()
	if len(sn) == 0 {
		fmt.Printf("Get node namespace failed.")
		// TODO: ?????? Error
		return nil, "", nil
	}

	return workloadSelector, sn, nil
}

func (i Invocation) ParseHost(srcNamespace, destination string) (string, error) {
	/* ????????? FQDN ?????????????????????????????????????????? host
	   - demo
	   - demo.default.svc.cluster.local
	   - demo:80
	   - demo.default.svc.cluster.local:80
	   - demo/ping
	   - demo.default.svc.cluster.local/ping
	   - demo:80/ping
	   - demo.default.svc.cluster.local:80/ping
	*/

	url := strings.Split(destination, "/")
	hostAndPort := strings.Split(strings.Trim(url[0], ":"), ":")
	if len(hostAndPort) <= 0 {
		fmt.Printf("There is no host or port in destination. %s", destination)
		return "", nil
	}
	// ???????????????????????????"."???
	hostPrefix := strings.Trim(strings.TrimSuffix(hostAndPort[0], FQDNSuffix), ".")
	if strings.Contains(hostPrefix, ".") {
		svcAndNamespace := strings.Split(hostPrefix, ".")
		if len(svcAndNamespace) <= 1 {
			fmt.Printf("There is no namespace in destination. %s", destination)
			return "", nil
		}
		desNamespace := svcAndNamespace[len(svcAndNamespace)-1]
		return desNamespace + "/" + hostPrefix + FQDNSuffix, nil
	} else {
		// ?????????????????????????????????????????????????????????
		return srcNamespace + "/" + hostPrefix + "." + srcNamespace + FQDNSuffix, nil
	}
}

func isSourceMatched(workload, labels map[string]string) bool {
	// ??????????????????, ???????????????????????????
	for k, v := range labels {
		w, ok := workload[k]
		if !ok {
			return false
		}
		if !strings.EqualFold(v, w) {
			return false
		}
	}
	return true
}

func (i *Invocation) syncLazySidecar(ctx *gin.Context, ls v1.LazySidecar, newHost string) error {
	mgr := *i.opts.Manager
	// patch := make([]utils.PatchEgressHost, 0)
	for _, egress := range ls.Spec.EgressHosts {
		if newHost == egress {
			return nil
		}
	}
	patch := utils.PatchEgressHost{
		Op:    "add",
		Path:  "/spec/egressHosts/-",
		Value: newHost,
	}
	patchByte, _ := json.Marshal(patch)

	return mgr.GetClient().Patch(ctx, &ls, client.RawPatch(types.JSONPatchType, patchByte))
}
