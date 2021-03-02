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

package server

import (
	"crypto/tls"

	"github.com/gorilla/mux"
)

// TunnelServer manages tunnels between itself and agents, receives requests
// from apiserver, and forwards requests to corresponding agents
type TunnelServer interface {
	Run() error
}

// NewTunnelServer returns a new TunnelServer
func NewTunnelServer(
	serverMasterAddr,
	serverAgentAddr string,
	serverCount int,
	tlsCfg *tls.Config,
	proxyStrategy string,
	udsName string) TunnelServer {
	ats := anpTunnelServer{
		serverMasterAddr: serverMasterAddr,
		serverAgentAddr:  serverAgentAddr,
		serverCount:      serverCount,
		tlsCfg:           tlsCfg,
		proxyStrategy:    proxyStrategy,
		udsName:          udsName,
	}
	return &ats
}

// ReverseProxyServer proxy request from tunnel agent to Kubernetes API Server
// which tunnel server located at
type ReverseProxyServer interface {
	Run() error
}

// NewReverseProxyServer returns a new ReverseProxyServer
func NewReverseProxyServer(address string, port int, tlsCfg *tls.Config) ReverseProxyServer {
	tlsClone := tlsCfg.Clone()
	// ProxyServer https only provide data encryption, auth will passthrough by real bankend
	tlsClone.ClientAuth = tls.RequestClientCert
	aps := reverseProxyServer{
		mux:     mux.NewRouter(),
		address: address,
		port:    port,
		tlsCfg:  tlsClone,
	}
	return &aps
}
