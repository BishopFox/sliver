package shm

import (
	"testing"
)

func TestShm(t *testing.T) {
	shmSize := 65536

	shmId, err := Get(IPC_PRIVATE, shmSize, IPC_CREAT|0777)
	if err != nil {
		t.Errorf("Get: %v", err)
	}

	data, err := At(shmId, 0, 0)
	if err != nil {
		t.Errorf("At: %v", err)
	}

	size, err := Size(shmId)
	if err != nil {
		t.Errorf("Size: %v", err)
	}

	if int(size) != shmSize {
		t.Errorf("Wrong size got %d expected %d", size, shmSize)
	}

	err = Rm(shmId)
	if err != nil {
		t.Errorf("Rm: %v", err)
	}

	err = Dt(data)
	if err != nil {
		t.Errorf("Dt: %v", err)
	}
}
