# tke-anp-tunnel

This repo aim to leverage Open Source project [ANP](https://github.com/kubernetes-sigs/apiserver-network-proxy/) and  [Yurt](https://github.com/openyurtio/openyurt/) to enable communication between global/mate cluster and managed cluster by a tunnel for `multi cloud` solution


## tke-anp-tunnel VS openyurt

1. Add `pull` mode support, agent can get data from global/mate cluster apiserver, so `tke-anp-tunnel` support both `pull` and `push` mode.

2. Enhance certificate management, the tunnel server/agent certificate managed by global/mate cluster apiserver by `CSR` and support certificate rotation.

3. Add reverse proxy to tunnel server so that `agent` add `add-on` service placed in managed cluster is able to access the service(such as norm) which in same flat network with tunnel server

4. Enhance the deploy topology：
 - Support **1** tunnel server works for **N** tunnel agent and **1** tunnel server works for **1** tunnel  agent for `public` cloud case and `private` cloud case
 - No `hostNetwork` dependency
 - Only tunnel server **MUST** exposed to the public network for agent join

4. Add hook plug-in mechanism/interface to execute customized logic support different user case

## Architecture

<img src="docs/img/arch.png" title="architecture">

## Build & Push image

1. Update image repo for your own
```
export REPO=huxl
```
2. Build and push docker image to docker hub
```
export REGION=cn && make release
```

## Deploy  topology  1: N

1. Deploy tunnel server on global/meta cluster


```
kubectl label nodes <node-name> openyurt.io/is-edge-worker=false
kubectl  create -f config/setup/yurt-tunnel-server.yaml
```

1. Deploy tunnel agent on managed cluster

```
kubectl  create -f config/setup/yurt-tunnel-agent.yaml
```

##  Hook

Currently, only support `PreStartTunnelAgent` and `PostStartTunnelAgent` hook to excute customized logic for different  cloud provider, for example:

1. Tunnel agent responsible for report the managed cluster  `admin` token to Tunnel server and persist it to global/meta cluster so that `tke-platform` use it to create k8s `ClientSet` to operator managed cluster,  `tke` and `tkestack` provider will implements different logic according to the case 

## HA
