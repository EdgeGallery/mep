# Copyright 2020-2021 Huawei Technologies Co., Ltd.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.14.2-alpine3.11 as builder

ENV HOME=/usr/mep

RUN apk update &&\
    apk add --no-cache libc-dev gcc shadow

RUN mkdir -p $HOME

WORKDIR /go/cache

ADD go.mod .
ADD go.sum .
RUN go mod download

COPY . $HOME

WORKDIR $HOME

RUN go build -buildmode=pie -ldflags '-linkmode "external" -extldflags "-static"' src/main/main.go

RUN chmod 550 $HOME/main &&\
    chmod -R 750 $HOME/conf &&\
    chmod 640 $HOME/conf/app_conf.yaml &&\
    chmod 500 -R $HOME/views &&\
    chmod 440 -R $HOME/views/error/404.tpl

FROM alpine:latest
RUN sed -i "s|umask 022|umask 027|g" /etc/profile

# Set the home directory to our app user's home.
ENV HOME=/usr/mep
ENV UID=166
ENV GID=166
ENV USER_NAME=eguser
ENV GROUP_NAME=eggroup
ENV ENV="/etc/profile"

# Create the home directory for the new app user.
RUN mkdir -p /usr/mep
RUN mkdir -p /usr/mep/bin

# Create an app user so our program doesn't run as root.
RUN apk update &&\
    apk add shadow &&\
    groupadd -r -g $GID $GROUP_NAME &&\
    useradd -r -u $UID -g $GID -d $HOME -s /sbin/nologin -c "Docker image user" $USER_NAME

## SETTING UP THE APP ##
WORKDIR $HOME

# Create log and cert directory
RUN chmod 750 $HOME &&\
    chmod 550 -R $HOME/bin &&\
    mkdir -p -m 750 $HOME/log &&\
    mkdir -p -m 750 $HOME/conf &&\
    mkdir -p -m 500 $HOME/views &&\
    chown -hR $USER_NAME:$GROUP_NAME $HOME

# Copy in the application exe.
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/main $HOME/bin
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/conf/ $HOME/conf/
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/views/ $HOME/views/

# Change to the app user.
USER $USER_NAME

ENTRYPOINT ["./bin/main"]