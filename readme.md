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

#### 1: Download the Proxmox VE ISO
[From here](https://www.proxmox.com/en/downloads)

#### 2: Mount the ISO locally

´
sudo rm -f -r ~/proxmox-iso
rm -f ~/Downloads/proxmox-ve_9.1-1.iso

curl --create-dirs -O --output-dir ~/Downloads https://enterprise.proxmox.com/iso/proxmox-ve_9.1-1.iso
mkdir ~/proxmox-iso
sudo mount -o loop ~/Downloads/proxmox-ve_9.1-1.iso /mnt
rsync -av /mnt/ ~/proxmox-iso/
sudo umount /mnt

# Build mbj-autoscaler and move it into ISO
go build -C ~/Documents/repos/mbj-autoscaler
sudo mkdir -p ~/proxmox-iso/rootfs/usr/local/bin/
sudo cp -r ~/Documents/repos/mbj-autoscaler ~/proxmox-iso/rootfs/usr/local/bin/
sudo chmod +x ~/proxmox-iso/rootfs/usr/local/bin/mbj-autoscaler

# Create service unit in ISO
sudo mkdir -p ~/proxmox-iso/rootfs/etc/systemd/system/multi-user.target.wants
sudo echo "
[Unit]
Description=MBJ Autoscaler
After=network.target

[Service]
ExecStart=/usr/local/bin/mbj-autoscaler
Restart=always
User=root

[Install]
WantedBy=multi-user.target" > ~/proxmox-iso/rootfs/etc/systemd/system/multi-user.target.wants/mbj-autoscaler.service
´

# Create first startup script
sudo echo "
#!/bin/bash

# Initial updates
apt update

# Install tools

## Ansible
apt install -y ansible

## kubeadm
apt install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
cat <<EOF >/etc/apt/sources.list.d/kubernetes.list
deb http://apt.kubernetes.io/ kubernetes-xenial main
EOF
apt update
apt install -y kubelet kubeadm

# Start services
systemctl start mbj-autoscaler" >  ~/proxmox-iso/rootfs/etc/rc.local/firstboot.sh
chmod +x ~/proxmox-iso/rootfs/etc/rc.local/firstboot.sh

# Create .kube config

sudo echo "
k8s credentials
" > ~/proxmox-iso/rootfs/root/.kube/config

cd ~/proxmox-iso
mkisofs -o ~/custom-proxmox.iso \
  -b isolinux/isolinux.bin \
  -c isolinux/boot.cat \
  -no-emul-boot \
  -boot-load-size 4 \
  -boot-info-table \
  -R -J -v -T .