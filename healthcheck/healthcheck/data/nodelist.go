/*
 * Copyright 2021 Huawei Technologies Co., Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package data

import (
	"errors"
)

type NodeList struct {
	NodeList map[string]bool
}

var EdgeList NodeList

func (c *NodeList) NewNodeList(ipList []string) NodeList {
	EdgeList = NodeList{
		NodeList: make(map[string]bool),
	}
	for _, edge := range ipList { //set EdgeList by HostList from MecM
		EdgeList.NodeList[edge] = false
	}
	return EdgeList
}

/*func (c *NodeList) SetNodeListFromMecM(m *controllers.MecMController) {

	for _, ip := range m.GetNodeIpList() {
		//TODO: check whether can insert key-value to map in this way?
		c.NodeList[ip] = false // default value for every checking edge is false
	}
}*/

func (c *NodeList) SetResult(ip string) error {
	_, ok := c.NodeList[ip]
	if ok { //c.NodeList contains this ip, which means this ip is in edge zone
		c.NodeList[ip] = true
		return nil
	} else {
		return errors.New("wrong ip: this ip is not in node list")
	}
}

func (c *NodeList) SetBadResult(ip string) error {
	_, ok := c.NodeList[ip]
	if ok { //c.NodeList contains this ip, which means this ip is in edge zone
		c.NodeList[ip] = false
		return nil
	} else {
		return errors.New("wrong ip: this ip is not in node list")
	}
}

/*func (c *NodeList) GetNodeList() map[string]bool {
	return EdgeList.NodeList
}*/
