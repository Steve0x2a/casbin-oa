// Copyright 2021 The casbin Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package object

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/casbin/casbin-oa/cloud"
)

func getMachineFromInstance(instance ecs.Instance) *Machine {
	machine := &Machine{
		Owner:       "admin",
		Name:        instance.InstanceName,
		CreatedTime: instance.CreationTime,
		Description: instance.Description,
		Ip:          instance.PublicIpAddress.IpAddress[0],
		Username:    "administrator",
		Password:    "123",
		AutoQuery:   true,
		IsPermanent: false,
		Services: []*Service{{
			No:             0,
			Name:           "casbintest",
			Path:           "C:/github_repos/casbintest",
			Port:           9095,
			ProcessId:      0,
			ExpectedStatus: "Running",
			Status:         "",
			SubStatus:      "",
			Message:        "",
		}},
	}
	return machine
}

func getMachinesFromInstances(instances []ecs.Instance) []*Machine {
	machines := []*Machine{}
	for _, instance := range instances {
		machine := getMachineFromInstance(instance)
		machines = append(machines, machine)
	}
	return machines
}

func syncImpermanentMachines() {
	deleteImpermanentMachines()
	instances := cloud.GetInstances()
	machines := getMachinesFromInstances(instances)
	AddMachines(machines)

	serverIdMap := cloud.GetVsgServerIdMap()
	for _, instance := range instances {
		serverId := instance.InstanceId
		if _, ok := serverIdMap[serverId]; !ok {
			cloud.AddServerToSlb(serverId, 9095)
		}
	}
}
