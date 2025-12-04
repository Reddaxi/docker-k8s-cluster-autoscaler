I ran the bottom script to generate the protos

protoc \
  -I ./cluster-autoscaler \
  -I ./cluster-autoscaler/vendor \
  --go_out=. \
  --go-grpc_out=. \
  ./mbj-autoscaler/externalgrpc.proto
  
  
  
  protoc \
  -I ./cluster-autoscaler \
  -I ./cluster-autoscaler/vendor \
  --go_out=../mbj-autoscaler \
  --go-grpc_out=../mbj-autoscaler \
  ./cluster-autoscaler/cloudprovider/externalgrpc/protos/externalgrpc.proto


# Overview

## What runs where?
### Horizontal Node Autoscaler (K8S cluster)
The config that tells K8S that it can spin up nodes on the serverrack.

### Serverrack (Laptop)
Boots into a customized Proxmox ISO.
The ISO is customized to contain:
- my mbj-autoscaler which starts as a service automatically
- Ansible (Which the mbj-autoscaler will talk to, to spin up nodes)
- kubeadm + long lived .kube credentials

### K8S Node VMs
These will be running plain ubuntu 


# Guides

## Customizing the Proxmox ISO

Run the ´create-iso-from-proxmox.sh´ script.