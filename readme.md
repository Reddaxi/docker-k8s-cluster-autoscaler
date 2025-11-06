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