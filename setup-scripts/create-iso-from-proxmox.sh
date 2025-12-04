echo "##### Downloading and unpacking Proxmox VE ISO #####"

echo "Deleting existing ISO and mount directories..."
sudo rm -f -r ~/proxmox-iso
rm -f ~/Downloads/proxmox-ve_9.1-1.iso

echo "Downloading Proxmox VE ISO..."
curl --create-dirs -O --output-dir ~/Downloads https://enterprise.proxmox.com/iso/proxmox-ve_9.1-1.iso

echo "Mount iso and copy all files to ~/proxmox-iso/"
mkdir ~/proxmox-iso
sudo mount -o loop ~/Downloads/proxmox-ve_9.1-1.iso /mnt
rsync -av /mnt/ ~/proxmox-iso/
sudo umount /mnt


echo "##### Building mbj-autoscaler go application #####"
echo "Building..."
# Build mbj-autoscaler and move it into ISO
go build -C ~/Documents/repos/mbj-autoscaler -o ~/Documents/repos/mbj-autoscaler/mbj-autoscaler .
sudo mkdir -p ~/proxmox-iso/rootfs/usr/local/bin/
sudo cp ~/Documents/repos/mbj-autoscaler/mbj-autoscaler ~/proxmox-iso/rootfs/usr/local/bin/
sudo chmod +x ~/proxmox-iso/rootfs/usr/local/bin/mbj-autoscaler

# Create service unit in ISO
echo "##### Creating linux unit service #####"
sudo mkdir -p ~/proxmox-iso/rootfs/etc/systemd/system/
sudo tee ~/proxmox-iso/rootfs/etc/systemd/system/mbj-autoscaler.service > /dev/null << 'EOF'
[Unit]
Description=MBJ Autoscaler
After=network.target

[Service]
ExecStart=/usr/local/bin/mbj-autoscaler
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

# Enable the service
sudo mkdir -p ~/proxmox-iso/rootfs/etc/systemd/system/multi-user.target.wants
sudo ln -sf ../mbj-autoscaler.service ~/proxmox-iso/rootfs/etc/systemd/system/multi-user.target.wants/

# Create first startup script
echo "##### Creating first startup script #####"
sudo echo "
#!/bin/bash

# Initial updates
apt update

# Install tools

## Ansible
apt install -y ansible

## kubeadm (using modern repository)
apt install -y apt-transport-https ca-certificates curl gpg
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' > /etc/apt/sources.list.d/kubernetes.list
apt update
apt install -y kubelet kubeadm kubectl

# Start services
systemctl enable mbj-autoscaler
systemctl start mbj-autoscaler" | sudo tee ~/proxmox-iso/rootfs/usr/local/bin/firstboot.sh > /dev/null
sudo chmod +x ~/proxmox-iso/rootfs/usr/local/bin/firstboot.sh

# Create .kube config
echo "##### Creating .kube config #####"
sudo mkdir -p ~/proxmox-iso/rootfs/root/.kube/
sudo tee ~/proxmox-iso/rootfs/root/.kube/config > /dev/null << 'EOF'
# TODO: Replace with actual k8s credentials
apiVersion: v1
kind: Config
clusters: []
users: []
contexts: []
current-context: ""
EOF

echo "##### Creating Custom ISO image #####"
# Fix ownership issues and make all files writable
sudo chown -R $(whoami):$(whoami) ~/proxmox-iso
chmod -R u+w ~/proxmox-iso/

# Create ISO in home directory (writable location)
cd ~/proxmox-iso

echo "Using xorriso to create ISO..."
xorriso -as mkisofs -o ~/custom-proxmox.iso \
  -b boot/grub/i386-pc/eltorito.img \
  -c boot/boot.cat \
  -no-emul-boot \
  -boot-load-size 4 \
  -boot-info-table \
  -eltorito-alt-boot \
  -e efi/boot/bootx64.efi \
  -no-emul-boot \
  -R -J -v -T . 2>&1


if [ $? -eq 0 ]; then
    echo "✅ Custom ISO created successfully at: ~/custom-proxmox.iso"
    ls -lh ~/custom-proxmox.iso
else
    echo "❌ Failed to create ISO"
    exit 1
fi
