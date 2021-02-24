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
	"fmt"
	"time"

	"github.com/openyurtio/openyurt/pkg/projectinfo"
	"github.com/openyurtio/openyurt/pkg/yurttunnel/constants"
	kubeutil "github.com/openyurtio/openyurt/pkg/yurttunnel/kubernetes"
	"github.com/openyurtio/openyurt/pkg/yurttunnel/pki"
	"github.com/openyurtio/openyurt/pkg/yurttunnel/pki/certmanager"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/apiserver-network-proxy/pkg/server"
)

// NewYurttunnelServerCommand creates a new yurttunnel-server command
func NewYurttunnelServerCommand(stopCh <-chan struct{}) *cobra.Command {
	o := NewYurttunnelServerOptions()

	cmd := &cobra.Command{
		Use:   "Launch " + projectinfo.GetServerName(),
		Short: projectinfo.GetServerName() + " sends requests to " + projectinfo.GetAgentName(),
		RunE: func(c *cobra.Command, args []string) error {
			if o.version {
				fmt.Printf("%s: %#v\n", projectinfo.GetServerName(), projectinfo.Get())
				return nil
			}
			fmt.Printf("%s version: %#v\n", projectinfo.GetServerName(), projectinfo.Get())

			if err := o.validate(); err != nil {
				return err
			}
			if err := o.complete(); err != nil {
				return err
			}
			if err := o.run(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&o.version, "version", o.version,
		fmt.Sprintf("print the version information of the %s.",
			projectinfo.GetServerName()))
	flags.StringVar(&o.kubeConfig, "kube-config", o.kubeConfig,
		"path to the kubeconfig file.")
	flags.StringVar(&o.bindAddr, "bind-address", o.bindAddr,
		fmt.Sprintf("the ip address on which the %s will listen.",
			projectinfo.GetServerName()))
	flags.StringVar(&o.insecureBindAddr, "insecure-bind-address", o.insecureBindAddr,
		fmt.Sprintf("the ip address on which the %s will listen without tls.",
			projectinfo.GetServerName()))
	flags.StringVar(&o.certDNSNames, "cert-dns-names", o.certDNSNames,
		"DNS names that will be added into server's certificate. (e.g., dns1,dns2)")
	flags.StringVar(&o.certIPs, "cert-ips", o.certIPs,
		"IPs that will be added into server's certificate. (e.g., ip1,ip2)")
	flags.IntVar(&o.serverCount, "server-count", o.serverCount,
		"The number of proxy server instances, should be 1 unless it is an HA server.")
	flags.StringVar(&o.proxyStrategy, "proxy-strategy", o.proxyStrategy,
		"The strategy of proxying requests from tunnel server to agent.")

	return cmd
}

// YurttunnelServerOptions has the information that required by the
// yurttunel-server
type YurttunnelServerOptions struct {
	kubeConfig               string
	bindAddr                 string
	insecureBindAddr         string
	certDNSNames             string
	certIPs                  string
	version                  bool
	serverAgentPort          int
	serverMasterPort         int
	serverMasterInsecurePort int
	serverCount              int
	serverAgentAddr          string
	serverMasterAddr         string
	clientset                kubernetes.Interface
	sharedInformerFactory    informers.SharedInformerFactory
	proxyStrategy            string
}

// NewYurttunnelServerOptions creates a new YurtNewYurttunnelServerOptions
func NewYurttunnelServerOptions() *YurttunnelServerOptions {
	o := &YurttunnelServerOptions{
		bindAddr:         "0.0.0.0",
		insecureBindAddr: "127.0.0.1",
		serverCount:      1,
		serverAgentPort:  constants.YurttunnelServerAgentPort,
		serverMasterPort: constants.YurttunnelServerMasterPort,
		proxyStrategy:    string(server.ProxyStrategyDestHost),
	}
	return o
}

// validate validates the YurttunnelServerOptions
func (o *YurttunnelServerOptions) validate() error {
	if len(o.bindAddr) == 0 {
		return fmt.Errorf("%s's bind address can't be empty",
			projectinfo.GetServerName())
	}
	return nil
}

// complete completes all the required options
func (o *YurttunnelServerOptions) complete() error {
	o.serverAgentAddr = fmt.Sprintf("%s:%d", o.bindAddr, o.serverAgentPort)
	o.serverMasterAddr = fmt.Sprintf("%s:%d", o.bindAddr, o.serverMasterPort)
	klog.Infof("server will accept %s requests at: %s, "+
		"server will accept master https requests at: %s "+
		projectinfo.GetAgentName(), o.serverAgentAddr)
	var err error
	// function 'kubeutil.CreateClientSet' will try to create the clientset
	// based on the in-cluster config if the kubeconfig is empty. As
	// yurttunnel-server will run on the cloud, the in-cluster config should
	// be available.
	o.clientset, err = kubeutil.CreateClientSet(o.kubeConfig)
	if err != nil {
		return err
	}
	o.sharedInformerFactory =
		informers.NewSharedInformerFactory(o.clientset, 10*time.Second)

	return nil
}

// run starts the yurttunel-server
func (o *YurttunnelServerOptions) run(stopCh <-chan struct{}) error {

	// 1. create a certificate manager for the tunnel server and run the
	// csr approver for both yurttunnel-server and yurttunnel-agent
	serverCertMgr, err :=
		certmanager.NewYurttunnelServerCertManager(
			o.clientset, o.certDNSNames, o.certIPs, stopCh)
	if err != nil {
		return err
	}
	serverCertMgr.Start()
	go certmanager.NewCSRApprover(o.clientset, o.sharedInformerFactory.Certificates().V1beta1().CertificateSigningRequests()).
		Run(constants.YurttunnelCSRApproverThreadiness, stopCh)

	// 2. generate the TLS configuration based on the latest certificate
	rootCertPool, err := pki.GenRootCertPool(o.kubeConfig,
		constants.YurttunnelCAFile)
	if err != nil {
		return fmt.Errorf("fail to generate the rootCertPool: %s", err)
	}
	tlsCfg, err :=
		pki.GenTLSConfigUseCertMgrAndCertPool(serverCertMgr, rootCertPool)
	if err != nil {
		return err
	}

	// 3. after all of informers are configured completed, start the shared index informer
	o.sharedInformerFactory.Start(stopCh)

	// 4. waiting for the certificate is generated
	_ = wait.PollUntil(5*time.Second, func() (bool, error) {
		// keep polling until the certificate is signed
		if serverCertMgr.Current() != nil {
			return true, nil
		}
		klog.Infof("waiting for the master to sign the %s certificate",
			projectinfo.GetServerName())
		return false, nil
	}, stopCh)

	// 5. start reverse proxy
	tcas := NewReverseProxyServer(
		o.bindAddr,
		constants.YurttunnelServerReversePorxyPort,
		tlsCfg,
	)
	if err := tcas.Run(); err != nil {
		return err
	}

	// 6. start the tunnel server
	ts := NewTunnelServer(
		o.serverMasterAddr,
		o.serverAgentAddr,
		o.serverCount,
		tlsCfg,
		o.proxyStrategy)
	if err := ts.Run(); err != nil {
		return err
	}

	<-stopCh
	return nil
}
