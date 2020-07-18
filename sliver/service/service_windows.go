package service

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
