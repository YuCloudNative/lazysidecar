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

package errcode

var (
	ErrorGetCustomCRDFail     = NewError(20010001, "获取自定义 CRD 失败")
	ErrorGetSidecarCRDFail    = NewError(20010002, "获取 Sidecar CRD 失败")
	ErrorUpdateSidecarCRDFail = NewError(20010003, "更新 Sidecar CRD 失败")
)
