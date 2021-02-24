/*
Copyright 2020 The OpenYurt Authors.

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

package tkestack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/openyurtio/openyurt/pkg/yurttunnel/constants"
	"github.com/openyurtio/openyurt/pkg/yurttunnel/hook/interfaces"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// AgentHook for execute
type AgentHook struct {
	ProviderName string
	ClusterName  string
	Clientset    kubernetes.Interface
}

type ThingSpec struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// ClusterCredential records the credential information needed to access the cluster.
type ClusterCredential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ClusterName       string  `json:"clusterName"`
	Token             *string `json:"token,omitempty"`
}

// ClusterCredentialList is the whole list of all ClusterCredential which owned by a tenant.
type ClusterCredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterCredential `json:"items"`
}

// NewTKEStackHookProvider creates a YurtCertificateManager
func NewTKEStackHookProvider(providerName string, clusterName string,
	clientset kubernetes.Interface) interfaces.TunnelHookProvider {
	return &AgentHook{
		ProviderName: providerName,
		ClusterName:  clusterName,
		Clientset:    clientset,
	}
}

// PreStartTunnelAgent excute pre start tunnel agent hook
func (hook *AgentHook) PreStartTunnelAgent() error {
	return nil
}

// PostStartTunnelAgent excute post start tunnel agent hook
func (hook *AgentHook) PostStartTunnelAgent() error {
	// steps:
	// 1. get cluster credential by cluster name
	// 2. patch token field to cluster credential
	if true {
		klog.Infof("excute PostStartTunnelAgent by tkestack provider")
		return nil
	}
	ccl := ClusterCredentialList{}
	restclient := hook.Clientset.Discovery().RESTClient()
	data, err := restclient.
		Get().
		AbsPath("/apis/platform.tkestack.io/v1/clustercredentials?fieldSelector=clusterName=" + hook.ClusterName).
		DoRaw()
	if err != nil {
		return fmt.Errorf("Failed to get cluster %s credential for cluster:%s", hook.ClusterName, err)
	}
	err = json.Unmarshal(data, &ccl)
	if err != nil {
		return err
	}
	if len(ccl.Items) == 0 {
		return fmt.Errorf("cluster cluster credential for cluster: %s is not ready", hook.ClusterName)
	}
	tokenByte, err := ioutil.ReadFile(constants.YurttunnelTokenFile)
	if err != nil {
		return fmt.Errorf("Failed to read token from %s: %s", constants.YurttunnelTokenFile, err)
	}

	things := make([]ThingSpec, 1)
	things[0].Op = "replace"
	things[0].Path = "/token"
	things[0].Value = string(tokenByte)

	_, err = restclient.Patch(types.MergePatchType).
		AbsPath("/apis/platform.tkestack.io/v1/clustercredentials/" + ccl.Items[0].Name).
		Body(things).
		DoRaw()
	if err != nil {
		return fmt.Errorf("patch cluster credential for cluster %s faild: %s", hook.ClusterName, err)
	}
	return nil
}
