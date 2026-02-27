echo "##### Downloading and unpacking Proxmox VE ISO #####"

echo "Deleting existing ISO and mount directories..."
sudo rm -f -r ~/proxmox-iso
rm -f ~/Downloads/proxmox-ve_9.1-1.iso

echo "Downloading Proxmox VE ISO..."
curl --create-dirs -O --output-dir ~/Downloads https://enterprise.proxmox.com/iso/proxmox-ve_9.1-1.iso

