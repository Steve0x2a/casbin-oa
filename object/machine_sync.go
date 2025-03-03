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
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/casbin/casbin-oa/ssh"
	"github.com/casbin/casbin-oa/util"
)

var reBatNames *regexp.Regexp
var machineMutex *sync.RWMutex

func init() {
	reBatNames = regexp.MustCompile(`\\Desktop\\(.*?)\.bat`)
	machineMutex = new(sync.RWMutex)
}

func getMachineService(id string, service *Service) *Service {
	machine := GetMachine(id)
	res := machine.Services[service.No]
	return res
}

func updateMachineService(id string, service *Service) bool {
	machineMutex.Lock()
	defer machineMutex.Unlock()

	machine := GetMachine(id)
	machine.Services[service.No] = service
	return UpdateMachine(id, machine)
}

func updateMachineServiceStatus(machine *Machine, service *Service, status string, subStatus string, message string) bool {
	id := machine.getId()
	service.Status = status
	service.SubStatus = subStatus
	service.Message = message
	return updateMachineService(id, service)
}

func parseBatName(s string) string {
	res := reBatNames.FindStringSubmatch(s)
	if res == nil {
		return ""
	}

	return res[1]
}

func getBatNamesFromOutput(output string) map[string]int {
	batNameMap := map[string]int{}

	output = strings.ReplaceAll(output, "\r", "")
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		tokens := strings.Split(line, " ")
		tokens2 := []string{}
		for _, token := range tokens {
			if token != "" {
				tokens2 = append(tokens2, token)
			}
		}

		if len(tokens2) < 5 || strings.ToLower(tokens2[0]) != `c:\windows\system32\cmd.exe` || tokens2[1] != "/c" {
			continue
		}

		batName := parseBatName(tokens2[2])
		processId := util.ParseInt(tokens2[len(tokens2)-1])
		batNameMap[batName] = processId
		//fmt.Printf("%s, %d\n", batName, processId)
	}

	return batNameMap
}

func (machine *Machine) runCommand(command string) string {
	output := ssh.RunCommand(machine.Ip, machine.Username, machine.Password, command)
	return output
}

func getBatInfo(machine *Machine) map[string]int {
	command := `wmic process where (name="cmd.exe") get CommandLine, ProcessID`
	output := machine.runCommand(command)
	batNameMap := getBatNamesFromOutput(output)
	return batNameMap
}

func doPull(machine *Machine, service *Service) error {
	updateMachineServiceStatus(machine, service, "Pull", "In Progress", "")

	command := fmt.Sprintf("cd C:/github_repos/%s && git pull --rebase --autostash", service.Name)
	output := machine.runCommand(command)

	var err error
	if !strings.Contains(output, "Applying autostash resulted in conflicts") && (strings.Contains(output, "Successfully rebased and updated") || strings.Contains(output, "Current branch master is up to date")) {
		err = nil
		updateMachineServiceStatus(machine, service, "Pull", "Done", "")
	} else {
		err = fmt.Errorf(output)
		updateMachineServiceStatus(machine, service, "Pull", "Error", output)
	}
	return err
}

func doBuild(machine *Machine, service *Service) error {
	updateMachineServiceStatus(machine, service, "Build", "In Progress", "")

	command := fmt.Sprintf("cd C:/github_repos/%s/web && yarn install", service.Name)
	output := machine.runCommand(command)

	var err error
	if strings.Contains(output, "Done in ") {
		err = nil
	} else {
		err = fmt.Errorf(output)
		updateMachineServiceStatus(machine, service, "Build", "Error", output)
		return err
	}

	command = fmt.Sprintf("cd C:/github_repos/%s/web && yarn build", service.Name)
	output = machine.runCommand(command)

	if strings.Contains(output, "Done in ") {
		err = nil
		updateMachineServiceStatus(machine, service, "Build", "Done", "")
	} else {
		err = fmt.Errorf(output)
		updateMachineServiceStatus(machine, service, "Build", "Error", output)
	}
	return err
}

func doDeploy(machine *Machine, service *Service) error {
	updateMachineServiceStatus(machine, service, "Deploy", "In Progress", "")

	command := fmt.Sprintf("cd C:/github_repos/%s/oss && go test", service.Name)
	output := machine.runCommand(command)

	if strings.Contains(output, "no required module provides package") {
		command2 := fmt.Sprintf("cd C:/github_repos/%s && go mod tidy", service.Name)
		output2 := machine.runCommand(command2)
		println(output2)

		if strings.Contains(output2, "error") {
			err := fmt.Errorf(output2)
			updateMachineServiceStatus(machine, service, "Build", "Error", output)
			return err
		}

		output = machine.runCommand(command)
	}

	var err error
	if strings.Contains(output, "PASS") && strings.Contains(output, "ok") {
		err = nil
		updateMachineServiceStatus(machine, service, "Deploy", "Done", "")
	} else {
		err = fmt.Errorf(output)
		updateMachineServiceStatus(machine, service, "Deploy", "Error", output)
	}
	return err
}

func doStart(machine *Machine, service *Service) error {
	updateMachineServiceStatus(machine, service, "Running", "In Progress", "")

	command1 := fmt.Sprintf(`SCHTASKS /Create /SC ONCE /ST "00:00" /TN "%s" /TR "CMD /C START '' 'C:\Users\Administrator\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup\%s.bat - 快捷方式.lnk' /K CD /D '%%CD%%'"`, service.Name, service.Name)
	command2 := fmt.Sprintf(`SCHTASKS /Run /TN "%s"`, service.Name)
	command3 := fmt.Sprintf(`SCHTASKS /Delete /TN "%s" /F`, service.Name)
	command := fmt.Sprintf("%s && %s && %s", command1, command2, command3)
	output := machine.runCommand(command)

	var err error
	if strings.Contains(output, "成功创建") && strings.Contains(output, "尝试运行") && strings.Contains(output, "被成功删除") {
		err = nil
		updateMachineServiceStatus(machine, service, "Running", "Done", "")
	} else {
		err = fmt.Errorf(output)
		updateMachineServiceStatus(machine, service, "Running", "Error", output)
	}
	return err
}

func doStop(machine *Machine, service *Service) error {
	updateMachineServiceStatus(machine, service, "Stopped", "In Progress", "")

	command := fmt.Sprintf("taskkill /T /F /PID %d", service.ProcessId)
	machine.runCommand(command)

	updateMachineServiceStatus(machine, service, "Stopped", "Done", "")
	return nil
}

func (machine *Machine) syncProcessIds() bool {
	affected := false
	batNameMap := getBatInfo(machine)
	for _, service := range machine.Services {
		if processId, ok := batNameMap[service.Name]; ok {
			if service.Status != "Running" || service.ProcessId != processId {
				affected = true
				service.Status = "Running"
				service.ProcessId = processId
			}
		} else {
			if service.Status != "Stopped" || service.ProcessId != processId {
				affected = true
				service.Status = "Stopped"
				service.ProcessId = -1
			}
		}
	}
	return affected
}

func GetProcessIdSyncedMachine(id string) *Machine {
	machineMutex.Lock()
	defer machineMutex.Unlock()

	machine := GetMachine(id)
	affected := machine.syncProcessIds()
	if affected {
		updateMachine(machine.Owner, machine.Name, machine)
	}
	return machine
}

func (machine *Machine) DoActions() {
	for _, service := range machine.Services {
		//doPull(machine, service)
		//doBuild(machine, service)
		//doDeploy(machine, service)
		if service.ExpectedStatus == "Running" && service.Status == "Stopped" {
			doStart(machine, service)
		} else if service.ExpectedStatus == "Stopped" && service.Status == "Running" {
			doStop(machine, service)
		}
	}

	machine.syncProcessIds()
}
