
## Feature

* listener与server完全解耦
* rust编写的模块化热插拔的implant


## Clone

`git clone --recurse-submodules https://github.com/chainreactors/malice-network`

## Build

generate protobuf

```bash
# client
protoc -I proto/ proto/client/clientpb/client.proto  --go_out=paths=source_relative:proto/

# implant
protoc -I proto/ proto/implant/commonpb/common.proto  --go_out=paths=source_relative:proto/
protoc -I proto/ proto/implant/pluginpb/plugin.proto  --go_out=paths=source_relative:proto/
```

``` bash
# listener
protoc -I proto/ proto/listener/lispb/listener.proto  --go_out=paths=source_relative:proto/


# rpc
protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
protoc -I proto/ proto/services/listenerrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
```


## Roadmap

### v0.0.1

- [ ] implant
  - [ ] scalability
    - [ ] pe
    - [ ] dll/so
    - [ ] bof
    - [ ] clr
  - [ ] basic function

- [ ] server
  - [ ] scalability
    - [ ] module, IoM internal module
    - [ ] alias, execute-assembly
    - [ ] extension, dll pe 
    - [ ] plugin , sdk 



## Thanks 

- [sliver](https://github.com/BishopFox/sliver) 从中参考并复用了大量的代码