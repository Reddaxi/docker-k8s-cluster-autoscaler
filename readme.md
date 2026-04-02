I ran the bottom script to generate the protos
  
  protoc \
  --go_out=../mbj-autoscaler \
  --go-grpc_out=../mbj-autoscaler \
  ./cluster-autoscaler/cloudprovider/externalgrpc/protos/externalgrpc.proto

Built with:

(cd ~/repos/homecluster/application-code/mbj-cluster-autoscaler ; sudo docker build --platform linux/arm64,linux/amd64 -t 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler . && \ 
sudo docker push 127.0.0.1:5000/infrastructure/iac/mbj-autoscaler)