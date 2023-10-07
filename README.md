
## Clone

`git clone --recurse-submodules https://github.com/chainreactors/malice-network`

## Build

generate protobuf

```bash
protoc -I proto/ proto/client/commonpb/common.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/client/clientpb/client.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/implant/implantpb/implant.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
```