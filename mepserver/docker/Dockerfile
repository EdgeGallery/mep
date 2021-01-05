# Copyright 2020 Huawei Technologies Co., Ltd.
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

FROM golang:1.14.2 as builder

ENV GOPROXY https://goproxy.io
ENV GO111MODULE on
ENV HOME=/usr/mep

RUN apt update &&\
    apt install libc-dev gcc

RUN mkdir -p $HOME

WORKDIR /go/cache

ADD go.mod .
ADD go.sum .
RUN go mod download

COPY . $HOME

WORKDIR $HOME

RUN GOOS=linux go build -buildmode=pie -ldflags '-linkmode "external" -extldflags "-static"' -o app

RUN chmod -R 550 $HOME/app &&\
    chmod -R 750 $HOME/conf &&\
    chmod 640 $HOME/conf/app.conf &&\
    chmod 550 $HOME/start.sh

FROM alpine:latest
RUN sed -i "s|umask 022|umask 027|g" /etc/profile
# Create the home directory for the new mep user.
RUN mkdir -p /usr/mep
RUN mkdir -p /usr/mep/bin

# Set the home directory to our app user's home.
ENV HOME=/usr/mep
ENV UID=166
ENV GID=166
ENV USER_NAME=eguser
ENV GROUP_NAME=eggroup
ENV ENV="/etc/profile"

# Create an app user so our program doesn't run as root.
RUN apk update &&\
    apk add shadow &&\
    groupadd -r -g $GID $GROUP_NAME &&\
    useradd -r -u $UID -g $GID -d $HOME -s /sbin/nologin -c "Docker image user" $USER_NAME

## SETTING UP THE APP ##
WORKDIR $HOME

RUN chmod 750 $HOME &&\
    chmod 550 -R $HOME/bin &&\
    mkdir -p -m 700 $HOME/ssl &&\
    mkdir -p -m 750 $HOME/log &&\
    mkdir -p -m 700 $HOME/keys  &&\
    mkdir -p -m 700 $HOME/cprop  &&\
    mkdir -p -m 700 $HOME/sprop  &&\
    mkdir -p -m 700 $HOME/wprop  &&\
    mkdir -p -m 700 $HOME/wnprop &&\
    mkdir -p -m 750 $HOME/conf &&\
    chown -hR $USER_NAME:$GROUP_NAME $HOME

# Copy in the application exe.
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/app $HOME/bin
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/conf/ $HOME/conf/
COPY --from=builder --chown=$USER_NAME:$GROUP_NAME $HOME/start.sh $HOME/bin

USER $USER_NAME

EXPOSE 8088

CMD ["sh", "-c", "$HOME/bin/start.sh"]