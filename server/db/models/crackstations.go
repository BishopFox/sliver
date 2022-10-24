package models

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
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// Crackstation - History of crackstation jobs
type Crackstation struct {
	// ID = crackstation name
	ID        string    `gorm:"primaryKey;->"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Tasks     []CrackTask
}

// BeforeCreate - GORM hook
func (c *Crackstation) BeforeCreate(tx *gorm.DB) (err error) {
	c.CreatedAt = time.Now()
	return nil
}

type CrackTask struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CrackstationID string
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	StartedAt      time.Time
	FinishedAt     time.Time
	Status         string

	Data []byte // protobuf
}

func (c *CrackTask) ToProtobuf() *clientpb.CrackTask {
	task := &clientpb.CrackTask{}
	proto.Unmarshal(c.Data, task)
	task.CrackTaskCreatedAt = c.CreatedAt.Unix()
	task.CrackTaskStartedAt = c.StartedAt.Unix()
	task.CrackTaskFinishedAt = c.FinishedAt.Unix()
	task.CrackTaskStatus = c.Status
	return task
}

// BeforeCreate - GORM hook
func (c *CrackTask) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	c.Status = "queued"
	return nil
}
