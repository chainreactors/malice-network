
## Clone

`git clone --recurse-submodules https://github.com/chainreactors/malice-network`

## Build

generate protobuf

```bash
protoc -I proto/ proto/commonpb/common.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/implantpb/implant.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/clientpb/client.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/services/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
```