# EdgeGallery MEP项目

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Jenkins](https://img.shields.io/jenkins/build?jobUrl=http%3A%2F%2Fjenkins.edgegallery.org%2Fview%2FMEC-PLATFORM-BUILD%2Fjob%2Fmep-docker-image-build-update-daily-master%2F)

## 介绍

Edgegallery MEP是依据ETSI MEC 003 [1]和011 [2]标准实现的MEP开源方案。

ETSI GS MEC 003中定义的MEP，提供了一个使应用程序可以发现、通告、使用和提供MEC服务的环境。在从MEC平台管理器、应用程序或服务接收到流量规则的更新，激活或停用后，MEP会进行对应的执行动作。MEP还从MEC平台管理器接收DNS记录，并使用它们来配置DNS代理/服务器。

通过ETSI GS MEC 011中定义的MEP和应用程序之间的Mp1参考点，可以启用基本功能，例如：

* MEC服务协同：
    - 提供和使用MEC服务的认证和授权；
    - MEC应用程序向MEC平台注册/注销其提供的MEC服务，并向MEC平台更新有关MEC服务可用性的变化；
    - 向相关的MEC应用程序通知MEC服务可用性的改变；
    - 发现可用的MEC服务；
* MEC应用协助：
    - MEC应用程序可用性订阅；
    - MEC应用程序终止订阅；

## MEP架构

MEP Mp1服务注册和发现基于servicecomb服务中心实现[3]。Servicecomb服务中心是一个基于Restful的服务注册表，提供微服务发现和微服务管理。MEP利用其注册表功能和插件机制来实现Mp1接口。mep-server模块是Mp1 API的MEP服务器的核心实现。提供了API，供MEC Apps在MEC平台中注册或发现服务。

## MEP代码目录
```
├── kong-plugin
│   ├── appid-header
│   └── kong.conf
├── mepauth
├── mepserver
└── README.md

```
上面是MEP项目的目录树：
- kong-plugin: mep api网关kong插件
- mepserver: mep server实现
- mepauth: mepauth模块为应用提供令牌申请api

## MEP构建和运行

大多数MEP项目代码是由golang开发的，kong插件是由lua开发的。MEP项目通过docker image发布。

### 建立mep-auth

```
cd mepauth
sudo ./docker-build.sh

```

### 建立mep服务器

```
cd mepauth
sudo ./docker-build.sh
```

### 运行mepauth

```
docker run -itd --name mepauth \
             --cap-drop All \
             --network mep-net \
             --link kong-service:kong-service \
             -v ${MEP_CERTS_DIR}/jwt_publickey:${MEPAUTH_KEYS_DIR}/jwt_publickey:ro \
             -v ${MEP_CERTS_DIR}/jwt_encrypted_privatekey:${MEPAUTH_KEYS_DIR}/jwt_encrypted_privatekey:ro \
             -v ${MEP_CERTS_DIR}/mepserver_tls.crt:${MEPAUTH_SSL_DIR}/server.crt:ro \
             -v ${MEP_CERTS_DIR}/mepserver_tls.key:${MEPAUTH_SSL_DIR}/server.key:ro \
             -v ${MEP_CERTS_DIR}/ca.crt:${MEPAUTH_SSL_DIR}/ca.crt:ro \
             -v ${MEPAUTH_CONF_PATH}:/usr/mep/mepauth.properties \
             -e "MEPAUTH_APIGW_HOST=kong-service" \
             -e "MEPAUTH_APIGW_PORT=8444"  \
             -e "MEPAUTH_CERT_DOMAIN_NAME=${DOMAIN_NAME}" \
             edgegallery/mepauth:latest
```

MEP_CERTS_DIR是放置mepauth服务器证书和密钥的位置。MEPAUTH_CONF_PATH是mepauth的配置文件。

### 运行mepserver
MEP_CERTS_DIR是放置mep服务器证书和密钥的目录。
```
docker run -itd --name mepserver --network mep-net -e "SSL_ROOT=${MEPSERVER_SSL_DIR}" \
                                 --cap-drop All \
                                 -v ${MEP_CERTS_DIR}/mepserver_tls.crt:${MEPSERVER_SSL_DIR}/server.cer:ro \
                                 -v ${MEP_CERTS_DIR}/mepserver_encryptedtls.key:${MEPSERVER_SSL_DIR}/server_key.pem:ro \
                                 -v ${MEP_CERTS_DIR}/ca.crt:${MEPSERVER_SSL_DIR}/trust.cer:ro \
                                 -v ${MEP_CERTS_DIR}/mepserver_cert_pwd:${MEPSERVER_SSL_DIR}/cert_pwd:ro \
                                 edgegallery/mep:latest
```


有关MEP的构建和安装的更多详细信息，请参阅 [HERE](https://gitee.com/edgegallery/docs/blob/master/MEP/EdgeGallery%E6%9C%AC%E5%9C%B0%E5%BC%80%E5%8F%91%E9%AA%8C%E8%AF%81%E6%9C%8D%E5%8A%A1%E8%AF%B4%E6%98%8E%E4%B9%A6.md).

## 参考
[1] https://www.etsi.org/deliver/etsi_gs/MEC/001_099/003/02.01.01_60/gs_MEC003v020101p.pdf

[2] https://www.etsi.org/deliver/etsi_gs/MEC/001_099/011/01.01.01_60/gs_MEC011v010101p.pdf

[3] https://github.com/apache/servicecomb-service-center
