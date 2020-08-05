-- Copyright 2020 Huawei Technologies Co., Ltd.
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

local BasePlugin = require "kong.plugins.base_plugin"
local jwt_decoder = require "kong.plugins.jwt.jwt_parser"


local kong = kong
local type = type
local re_gmatch = ngx.re.gmatch

local AddAppIdHeaderHandler = {}


AddAppIdHeaderHandler.VERSION  = "1.0.0"
AddAppIdHeaderHandler.PRIORITY = 10


local function retrieve_token()
  local request_headers = kong.request.get_headers()
  local token_header = request_headers["authorization"]
  if token_header then
    if type(token_header) == "table" then
      token_header = token_header[1]
    end
    local iterator, iter_err = re_gmatch(token_header, "\\s*[Bb]earer\\s+(.+)")
    if not iterator then
      kong.log.err(iter_err)
    end

    local m, err = iterator()
    if err then
      kong.log.err(err)
    end

    if m and #m >0 then
      return m
    end
  end
end


local function add_app_id_check_ip()
  local token, err = retrieve_token()
  if err then
    kong.log.err(err)
    return kong.response.exit(500, { message = "Unexpected error." })
  end

  local jwt, err = jwt_decoder:new(token[1])
  token[1] = nil
  unpack(token)
  if err then
    kong.log.err(err)
    return kong.response.exit(500, { message = "Unexpected error."})
  end

  local claims = jwt.claims

  local app_id = claims["sub"]

  -- check client ip same
  local remote_addr = ngx.var.remote_addr

  local client_ip = claims["clientip"]

  if client_ip == "UNKNOWN_IP" or remote_addr ~= client_ip then
    return false
  end

  local set_header = kong.service.request.set_header
  local clear_header = kong.service.request.clear_header
  clear_header("X-AppinstanceID")
  set_header("X-AppinstanceID", app_id)
  return true
end


function AddAppIdHeaderHandler:access(conf)
  local ok, err = add_app_id_check_ip()
  if err then
    kong.log.err(err)
    return kong.response.exit(500, { message = "Unexpected error."})
  end
  if not ok then
    return kong.response.exit(401, { message = "Unauthorized" })
  end
end

return AddAppIdHeaderHandler
