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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/openyurtio/openyurt/pkg/yurttunnel/constants"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server"
	anpserver "sigs.k8s.io/apiserver-network-proxy/pkg/server"
	anpagent "sigs.k8s.io/apiserver-network-proxy/proto/agent"
)

var udsListenerLock sync.Mutex

// anpTunnelServer implements the TunnelServer interface using the
// apiserver-network-proxy package
type anpTunnelServer struct {
	serverMasterAddr         string
	serverMasterInsecureAddr string
	serverAgentAddr          string
	serverCount              int
	tlsCfg                   *tls.Config
	proxyStrategy            string
	udsName                  string
}

var _ TunnelServer = &anpTunnelServer{}

// Run runs the yurttunnel-server
func (ats *anpTunnelServer) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	var masterServerErr error
	defer cancel()

	proxyServer := anpserver.NewProxyServer(uuid.New().String(),
		[]anpserver.ProxyStrategy{anpserver.ProxyStrategy(ats.proxyStrategy)},
		ats.serverCount,
		&anpserver.AgentTokenAuthenticationOptions{})

	// 2. start the master server
	if ats.udsName != "" {
		masterServerErr = runUDSMasterServer(
			ctx,
			proxyServer,
			ats.udsName,
		)
	} else {
		masterServerErr = runMTLSMasterServer(
			ats.serverMasterAddr,
			ats.tlsCfg,
			proxyServer)
	}
	if masterServerErr != nil {
		return fmt.Errorf("fail to run master server: %s", masterServerErr)
	}
	// 3. start the agent server
	agentServerErr := runAgentServer(ats.tlsCfg, ats.serverAgentAddr, proxyServer)
	if agentServerErr != nil {
		return fmt.Errorf("fail to run agent server: %s", agentServerErr)
	}

	return nil
}

// runMTLSMasterServer runs an https server to handle requests from apiserver
func runMTLSMasterServer(
	masterServerAddr string,
	tlsCfg *tls.Config,
	s *server.ProxyServer) error {
	go func() {
		klog.Infof("start handling https request from master at %s", masterServerAddr)
		server := &http.Server{
			Addr:      masterServerAddr,
			TLSConfig: tlsCfg,
			Handler: &server.Tunnel{
				Server: s,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("failed to serve https request from master: %v", err)
		}
	}()
	return nil
}

func runUDSMasterServer(
	ctx context.Context,
	s *server.ProxyServer,
	udsName string) error {
	go func() {
		server := &http.Server{
			Handler: &server.Tunnel{
				Server: s,
			},
		}
		udsListener, err := getUDSListener(ctx, udsName)
		if err != nil {
			klog.ErrorS(err, "failed to get uds listener")
		}
		defer func() {
			udsListener.Close()
		}()
		err = server.Serve(udsListener)
		if err != nil {
			klog.ErrorS(err, "failed to serve uds requests")
		}

	}()
	return nil

}

// runAgentServer runs a grpc server that handles connections from the yurttunel-agent
// NOTE agent server is responsible for managing grpc connection yurttunel-server
// and yurttunnel-agent, and the proxy server is responsible for redirecting requests
// to corresponding yurttunel-agent
func runAgentServer(tlsCfg *tls.Config,
	agentServerAddr string,
	proxyServer *anpserver.ProxyServer) error {
	serverOption := grpc.Creds(credentials.NewTLS(tlsCfg))

	ka := keepalive.ServerParameters{
		// Ping the client if it is idle for `Time` seconds to ensure the
		// connection is still active
		Time: constants.YurttunnelANPGrpcKeepAliveTimeSec * time.Second,
		// Wait `Timeout` second for the ping ack before assuming the
		// connection is dead
		Timeout: constants.YurttunnelANPGrpcKeepAliveTimeoutSec * time.Second,
	}

	grpcServer := grpc.NewServer(serverOption,
		grpc.KeepaliveParams(ka))

	anpagent.RegisterAgentServiceServer(grpcServer, proxyServer)
	listener, err := net.Listen("tcp", agentServerAddr)
	klog.Info("start handling connection from agents")
	if err != nil {
		return fmt.Errorf("fail to listen to agent on %s: %s", agentServerAddr, err)
	}
	go grpcServer.Serve(listener)
	return nil
}

func getUDSListener(ctx context.Context, udsName string) (net.Listener, error) {
	udsListenerLock.Lock()
	defer udsListenerLock.Unlock()
	oldUmask := syscall.Umask(0007)
	defer syscall.Umask(oldUmask)
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "unix", udsName)
	if err != nil {
		return nil, fmt.Errorf("failed to listen(unix) name %s: %v", udsName, err)
	}
	return lis, nil
}
