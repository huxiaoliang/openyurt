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
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"k8s.io/klog/v2"
)

type reverseProxyServer struct {
	address string
	port    int
	tlsCfg  *tls.Config
}

// currently, only support agent access to global/meta cluster apiserver
// future will support norm api
type reverseProxyHandler struct {
	reverseProxy string
}

var _ ReverseProxyServer = &reverseProxyServer{}

func (o *reverseProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remote, err := url.Parse(o.reverseProxy)
	if err != nil {
		panic(err)
	}
	reverseProxy := httputil.NewSingleHostReverseProxy(remote)
	transport := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	reverseProxy.Transport = transport
	reverseProxy.FlushInterval = 100 * time.Millisecond
	reverseProxy.ServeHTTP(w, r)
}

func (o *reverseProxyServer) Run() error {
	// get apiserver address from env
	reverseProxy := "https://" +
		net.JoinHostPort(os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT"))
	hander := &reverseProxyHandler{reverseProxy: reverseProxy}
	go func() {
		server := http.Server{
			Addr:         fmt.Sprintf("%s:%d", o.address, o.port),
			Handler:      hander,
			TLSConfig:    o.tlsCfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		if err := server.ListenAndServeTLS("", ""); err != nil {
			klog.Errorf("failed to serve https request from master on %s:%d: %v", o.address, o.port, err)
		}
	}()
	klog.Infof("start handling apiserver proxy request from master at %s:%d", o.address, o.port)
	return nil
}
