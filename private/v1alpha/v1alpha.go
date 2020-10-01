package v1alpha

//go:generate protoc -I=. -I=$GOPATH/src --gogo_out=plugins=grpc:. --grpc-gateway_out=grpc_api_configuration=api.yaml:. --swagger_out=grpc_api_configuration=api.yaml:. api.proto
