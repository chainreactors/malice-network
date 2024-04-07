go mod tidy

go install google.golang.org/protobuf/cmd/protoc-gen-go
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

# client
protoc -I proto/ proto/client/clientpb/client.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/client/rootpb/root.proto  --go_out=paths=source_relative:proto/
# implant
protoc -I proto/ proto/implant/implantpb/implant.proto  --go_out=paths=source_relative:proto/


# listener
protoc -I proto/ proto/listener/lispb/listener.proto  --go_out=paths=source_relative:proto/


# rpc
protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
protoc -I proto/ proto/services/listenerrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
