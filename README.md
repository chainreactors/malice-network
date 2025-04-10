# Malice Network

blog posts:

- [v0.0.1 next generation C2 project](/wiki/blog/2024/08/16/%E4%B8%80%E4%B8%8B%E4%BB%A3c2%E8%AE%A1%E5%88%92-----internal-of-malice/)
- [v0.0.2 the Real Beginning](/wiki/blog/2024/09/23/IoM_v0.0.2/)
- [v0.0.3 RedTeam Infra&C2 framework](/wiki/blog/2024/11/20/IoM_v0.0.3/)
- [v0.0.4 Bootstrapping](/wiki/blog/2025/01/02/IoM_v0.0.4/)

## wiki

see: https://chainreactors.github.io/wiki/IoM/

implant: https://github.com/chainreactors/malefic

protocol: https://github.com/chainreactors/proto

## Roadmap

https://chainreactors.github.io/wiki/IoM/roadmap/

## Showcases

<summary>console</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/completion.gif"/>

<summary>login</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/login.gif"/>

<summary>tcp</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/tcp.gif"/>

<summary>website</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/website.gif"/>

<summary>execute_exe</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/execute_exe.gif"/>

<summary>load_addon</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/load_addon.gif"/>

<summary>armory</summary>
<img src="https://github.com/chainreactors/wiki/blob/master/docs/IoM/assets/armory.gif"/>

## Dependency

```bash
scoop install protobuf

go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
```

## Thanks

- [sliver](https://github.com/BishopFox/sliver) 从中参考并复用了大量的代码
