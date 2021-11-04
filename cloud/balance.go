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

package cloud

import (
	"fmt"
	"math"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func getTargetInstanceCount(slbRate int) int {
	res := int(math.Floor(float64(slbRate - 1) / 100.0)) + 1

	res = res - 3
	if res < 0 {
		res = 0
	}

	//fmt.Printf("target instance count = %d\n", res)
	return res
}

func getNewInstanceName(instanceCount int) string {
	return fmt.Sprintf("auto-%02d", instanceCount+1)
}

func getInstanceIdAndCreationTimeFromInstances(instances []ecs.Instance, instanceName string) (string, string) {
	for _, instance := range instances {
		if instance.InstanceName == instanceName {
			return instance.InstanceId, instance.CreationTime
		}
	}
	return "", ""
}

func getPassedMinutes(creationTime string) int {
	t, _ := time.Parse("2006-01-02T15:04Z", creationTime)
	duration := time.Since(t)
	return int(duration.Minutes())
}

func doBalance() {
	instances := GetInstances()
	instanceCount := len(instances)

	slbRate := GetSlbPacketRate()
	//slbRate = 60
	if slbRate == -1 {
		return
	}

	targetInstanceCount := getTargetInstanceCount(slbRate)
	if instanceCount == targetInstanceCount {
		fmt.Printf("instance_count: [%d] == target_instance_count: [%d], no change\n", instanceCount, targetInstanceCount)
	} else if instanceCount < targetInstanceCount {
		fmt.Printf("instance_count: [%d] < target_instance_count: [%d], will add instance..\n", instanceCount, targetInstanceCount)

		newInstanceName := getNewInstanceName(instanceCount)
		AddInstance(newInstanceName)
	} else {
		lastInstanceName := getNewInstanceName(instanceCount - 1)
		lastInstanceId, lastInstanceCreationTime := getInstanceIdAndCreationTimeFromInstances(instances, lastInstanceName)

		passedMinutes := getPassedMinutes(lastInstanceCreationTime)
		if passedMinutes < coolDownMinutes {
			fmt.Printf("instance_count: [%d] > target_instance_count: [%d], will keep instance because cooldown minutes: [%d] < [%d]\n", instanceCount, targetInstanceCount, passedMinutes, coolDownMinutes)
		} else {
			fmt.Printf("instance_count: [%d] > target_instance_count: [%d], will delete instance..\n", instanceCount, targetInstanceCount)

			DeleteInstance(lastInstanceId, lastInstanceName)
		}
	}
}
