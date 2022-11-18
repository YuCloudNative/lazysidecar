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

package routers

import (
	"github.com/gin-gonic/gin"

	"github.com/yucloudnative/lazysidecar/internal/routers/options"
	v1 "github.com/yucloudnative/lazysidecar/internal/routers/v1"
)

func NewRouter(opts *options.Options) *gin.Engine {
	r := gin.Default()

	invocation := v1.NewInvocation(opts)
	apiv1 := r.Group("/api/v1")
	{
		apiv1.POST("/invocation", invocation.Report)
	}

	return r
}
