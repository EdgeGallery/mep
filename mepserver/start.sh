#!/bin/sh
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

set +e

# Contain at most 63 characters
# Contain only lowercase alphanumeric characters or '-'
# Start with an alphanumeric character
# End with an alphanumeric character
validate_host_name()
{
 hostname="$1"
 len="${#hostname}"
 if [ "${len}" -gt "253" ] ; then
   return 1
 fi
 if ! echo "$hostname" | grep -qE '^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])(\.([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]{0,61}[a-zA-Z0-9]))*$' ; then
   return 1
 fi
 return 0
}

validate_host_name $(hostname)
valid_host_name="$?"
if [ ! "$valid_host_name" -eq "0" ] ; then
   echo "invalid host name"
   exit 1
fi


sed -i "s/^httpaddr.*=.*$/httpaddr = $(hostname)/g" conf/app.conf
set -e
umask 0027
$HOME/bin/app
