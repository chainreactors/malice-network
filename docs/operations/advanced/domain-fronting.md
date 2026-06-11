---
title: CDN 前置
---

# 使用 Cloudflare 进行 CDN 前置

IOM内置了HTTPS, 所以你可以使用CDN、云函数等来隐藏IOM服务

一些你必须满足的前提条件:

1. 一台具备公网IP的服务器, 用于部署/转发IOM服务端的端口
2. 一个Cloudflare账户, 用于解析域名
3. 一个域名，可以托管到Cloudflare
4. 确保你的pipline运行在80/443/8080/8443等cloudflare允许代理的端口上

## 绑定域名&IP
首先绑定一个到你公网服务器的A记录，并确保开启了代理状态，参考下图
![img_13.png](../../assets/advance/usage/domain_front/bind_host_ip.png)

## 申请cloudflare证书
需要绑定cloudflare下发的证书才能cdn的生效，申请后的证书存储到本地
![img_14.png](../../assets/advance/usage/domain_front/save_cert_key.png)

## 关闭缓存状态
![img_2.png](../../assets/advance/usage/domain_front/close_cache.png)

## 配置ssl/tls加密模式

请将ssl/tls加密模式设置为完全(请勿使用灵活)
![img_1.png](../../assets/advance/usage/domain_front/set_encrypt.png)

![img.png](../../assets/advance/usage/domain_front/set_encrypt_2.png)

## 服务器 部署&配置

上述流程完成后你需要在服务器上配置证书并启动，参考如下配置
```angular2html
  http:
  - enable: true
    encryption:
    - enable: true
      key: maliceofinternal
      type: aes
    - enable: true
      key: maliceofinternal
      type: xor
    error_page: ""
    host: 0.0.0.0
    name: http
    parser: auto
    port: 443
    tls:
      enable: true
      ca_file: cert/a.crt
      cert_file: cert/a.crt
      key_file: cert/a.key
```
然后启动server: 
![img_9.png](../../assets/advance/usage/domain_front/start_server.png)

## 编译&测试

根据上述步骤配置域名后，编译malefic上线即可
```
build beacon --profile http_default --address demo.cdn.com --target x86_64-pc-windows-gnu
```

部分日志截图参考如下:

server端:

![img_6.png](../../assets/advance/usage/domain_front/server_log.png)

Client端:

![img_8.png](../../assets/advance/usage/domain_front/client_log.png)

Malefic:

![img_10.png](../../assets/advance/usage/domain_front/malefic_log.png)
