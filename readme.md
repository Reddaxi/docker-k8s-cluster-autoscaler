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

k port-forward -n docker-registry svc/docker-registry 5000:5000


sudo docker build --platform linux/arm64 -t 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler . && \ 
sudo docker push 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler


Went through a whole thing when building;

Ran once to install QEMU handlers into kernel, whatever that means:
docker run --privileged --rm tonistiigi/binfmt --install all

Made a buildx builder:
docker buildx create --name multiarch --driver docker-container --use
docker buildx inspect --bootstrap

Then built with:
sudo docker build --platform linux/arm64 -t 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler . && \ 
sudo docker push 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler