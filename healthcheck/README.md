# Mep-Agent

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
![Jenkins](https://img.shields.io/jenkins/build?jobUrl=http://jenkins.edgegallery.org/view/mep/job/mep-agent-docker-build-master/)

## Introduction
Mep-Agent is a middleware that provides proxy services for third-party apps. It can help apps, which do not implement the ETSI interface to register to MEP, and realize app service registration and discovery.
Mep-Agent will start at the same time as the application container, and read the content in the file conf/app_instance_info.yaml to automatically register the service.

## MEP-Agent code directory

```
├─conf
├─docker
├─src
│  ├─config
│  ├─controllers
│  ├─main
│  ├─model
│  ├─router
│  ├─service
│  ├─test
│  └─util
└─views
    └─error
```

Above is the directory tree of MEP-Agent project, their usage is as belows:
- conf: mep-agent config file 
- docker: dockerfile file
- src: source code
  - config: config files
  - controllers: controller class
  - main: main method
  - model: model definition
  - router: route info
  - service: service logic
  - test: unit test
  - util: util tool file
- views: pages

## Build & Run

Mep-Agent is developed by the Go language and provides services in the form of a docker image. When it starts, it will read the configuration file and register the App to the MEP to realize service registration and discovery.

- ### Build

    git clone from mep-agent master repo
    ```
    git clone https://gitee.com/edgegallery/mep-agent.git
    ```
  
    build the mep-agent image
    ```
    docker build -t mep-agent:latest -f docker/Dockerfile .
    ```
  
- ### Run

    Prepare the certificate files and mepagent.properties, which contains ACCESS_KEY and SECRET_KEY, and run with
    ```
    docker run -itd --name mepagent \
      --cap-drop All \
      -e MEP_IP=<host IP> \ # host IP 为mep部署环境的IP地址
      -e MEP_APIGW_PORT=8443 \
      -e MEP_AUTH_ROUTE=mepauth \
      -e ENABLE_WAIT=true \
      -e AK=QVUJMSUMgS0VZLS0tLS0 \
      -e SK=DXPb4sqElKhcHe07Kw5uorayETwId1JOjjOIRomRs5wyszoCR5R7AtVa28KT3lSc \
      -e APPINSTID=5abe4782-2c70-4e47-9a4e-0ee3a1a0fd1f \
      -v /home/EG-LDVS/mepserver/ca.crt:/usr/mep/ssl/ca.crt:ro \
      -e "CA_CERT=/usr/mep/ssl/ca.crt" \
      -e "CA_CERT_DOMAIN_NAME=edgegallery" \
      -v /tmp/mepagent-conf/app_conf.yaml:/usr/mep/conf/app\_conf.yaml:ro \
      -v /home/EG-LDVS/mep-agent/conf/app_instance_info.yaml:/usr/mep/conf/app_instance_info.yaml:ro\ #可选， mep-agent默认自带一份样例app_instance_info.yaml用于注册
      edgegallery/mep-agent:latest
    ```

More details of the building and installation process please refer to [HERE](https://gitee.com/edgegallery/docs/blob/master/Projects/MEP/EdgeGallery%E6%9C%AC%E5%9C%B0%E5%BC%80%E5%8F%91%E9%AA%8C%E8%AF%81%E6%9C%8D%E5%8A%A1%E8%AF%B4%E6%98%8E%E4%B9%A6.md).
  
## Notice

Mep-Agent is written in Go language. In order to minimize the image, it adopts the process of statically compiling and then packaging, without relying on the basic Go language image, which greatly reduces the size of the image.
