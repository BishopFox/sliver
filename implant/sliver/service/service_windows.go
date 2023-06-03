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
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
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
