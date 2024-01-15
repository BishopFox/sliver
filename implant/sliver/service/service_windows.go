package service

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"strings"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	ReadOnlyServiceManagerPermissions = windows.SC_MANAGER_ENUMERATE_SERVICE
	ReadOnlyServicePermissions        = windows.SERVICE_QUERY_CONFIG | windows.SERVICE_QUERY_STATUS
	ConnectServiceManagerPermissions  = windows.SC_MANAGER_CONNECT
	StartServiceAndVerifyPermissions  = windows.SERVICE_START | windows.SERVICE_QUERY_STATUS
)

func StartService(hostname string, binPath string, arguments string, serviceName string, serviceDesc string) error {
	manager, err := mgr.ConnectRemote(hostname)
	if err != nil {
		return err
	}

	service, err := manager.CreateService(serviceName, binPath, mgr.Config{
		ErrorControl:   mgr.ErrorNormal,
		BinaryPathName: binPath,
		Description:    serviceDesc,
		DisplayName:    serviceName,
		ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
		StartType:      mgr.StartManual,
	}, arguments)

	if err != nil {
		return err
	}
	err = service.Start()
	if err != nil {
		return err
	}
	return err
}

func StopService(hostname string, serviceName string) error {
	manager, err := mgr.ConnectRemote(hostname)
	if err != nil {
		return err
	}
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return err
	}
	status, err := service.Control(svc.Stop)
	if err != nil {
		return err
	}
	timeout := time.Now().Add(10 * time.Second)

	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", svc.Stopped)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = service.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil

}

func RemoveService(hostname string, serviceName string) error {
	manager, err := mgr.ConnectRemote(hostname)
	if err != nil {
		return err
	}
	service, err := manager.OpenService(serviceName)
	if err != nil {
		return err
	}
	err = service.Delete()
	return err
}

/*
	Currently, golang.org/x/sys/windows/svc/mgr attempts to open the service
	manager with elevated permissions: https://github.com/golang/go/issues/51465
	https://github.com/golang/sys/blob/master/windows/svc/mgr/mgr.go#L34

	We should not need elevated privileges to list services, so we will have to
	define a custom service manager that asks for lower rights.

	Similarly, we need a custom OpenService function that does not ask for
	elevated rights so we can get the status and config of the service.
*/

func connectToServiceManager(hostname string, permissions uint32) (*mgr.Mgr, error) {
	/*
		According to Win32 API docs, if the machine name passed to OpenSCManager
		is NULL or empty, a connection will be opened to the local machine
		https://learn.microsoft.com/en-us/windows/win32/api/winsvc/nf-winsvc-openscmanagera
	*/
	connectHost := windows.StringToUTF16Ptr(hostname)
	handle, err := windows.OpenSCManager(connectHost, nil, permissions)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: handle}, nil
}

func openService(svcManager *mgr.Mgr, serviceName string, permissions uint32) (*mgr.Service, error) {
	handle, err := windows.OpenService(svcManager.Handle, windows.StringToUTF16Ptr(serviceName), permissions)
	if err != nil {
		return nil, err
	}
	return &mgr.Service{Name: serviceName, Handle: handle}, nil
}

func buildServiceDetail(serviceName string, config mgr.Config) *sliverpb.ServiceDetails {
	detail := &sliverpb.ServiceDetails{Name: serviceName}

	detail.DisplayName = config.DisplayName
	detail.Description = config.Description
	detail.StartupType = config.StartType
	detail.BinPath = config.BinaryPathName
	detail.Account = config.ServiceStartName
	// Will hopefully be filled in later
	detail.Status = 0

	return detail
}

func ListServices(hostName string) ([]*sliverpb.ServiceDetails, error) {
	var servicesList []*sliverpb.ServiceDetails
	var serviceErrors []string
	var operationError error

	manager, err := connectToServiceManager(hostName, ReadOnlyServiceManagerPermissions)
	if err != nil {
		return nil, err
	}
	defer manager.Disconnect()
	services, err := manager.ListServices()
	if err != nil {
		return nil, err
	}

	for _, serviceName := range services {
		serviceHandle, err := openService(manager, serviceName, ReadOnlyServicePermissions)
		if err != nil {
			serviceErrors = append(serviceErrors, fmt.Sprintf("%s: %s", serviceName, err.Error()))
			continue
		}
		serviceConfig, err := serviceHandle.Config()
		if err != nil {
			serviceErrors = append(serviceErrors, fmt.Sprintf("%s: %s", serviceName, err.Error()))
			continue
		}
		serviceInfo := buildServiceDetail(serviceName, serviceConfig)
		serviceStatus, err := serviceHandle.Query()
		if err != nil {
			serviceErrors = append(serviceErrors, err.Error())
			servicesList = append(servicesList, serviceInfo)
			continue
		}
		serviceInfo.Status = uint32(serviceStatus.State)
		servicesList = append(servicesList, serviceInfo)
	}

	if len(serviceErrors) > 0 {
		operationError = fmt.Errorf("%s", strings.Join(serviceErrors, "\n"))
	} else {
		operationError = nil
	}

	return servicesList, operationError
}

func GetServiceDetail(hostName string, serviceName string) (*sliverpb.ServiceDetails, error) {
	manager, err := connectToServiceManager(hostName, ReadOnlyServiceManagerPermissions)
	if err != nil {
		return nil, err
	}
	defer manager.Disconnect()
	serviceHandle, err := openService(manager, serviceName, ReadOnlyServicePermissions)
	if err != nil {
		return nil, err
	}
	serviceConfig, err := serviceHandle.Config()
	if err != nil {
		return nil, err
	}
	serviceDetail := buildServiceDetail(serviceName, serviceConfig)
	serviceStatus, err := serviceHandle.Query()
	if err != nil {
		serviceDetail.Status = 0
		// Even though we encountered an error, it was not fatal (we still got some information about the service)
		return serviceDetail, fmt.Errorf("Unknown (could not retrieve: %s)", err.Error())
	}
	serviceDetail.Status = uint32(serviceStatus.State)
	return serviceDetail, nil
}

func StartServiceByName(hostName string, serviceName string) error {
	manager, err := connectToServiceManager(hostName, ConnectServiceManagerPermissions)
	if err != nil {
		return err
	}

	serviceHandle, err := openService(manager, serviceName, StartServiceAndVerifyPermissions)
	if err != nil {
		return err
	}

	err = serviceHandle.Start()
	if err != nil {
		return err
	}
	timeout := time.Now().Add(10 * time.Second)
	status, err := serviceHandle.Query()
	if err != nil {
		return fmt.Errorf("could not retrieve service status: %v", err)
	}

	for status.State != svc.Running {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", svc.Running)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = serviceHandle.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}
