package prelude

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
	"crypto/md5"
	"encoding/hex"
	"io"
	"sort"
)

const (
	ErrorExitStatus   = -1
	SuccessExitStatus = 0
)

type Configuration interface {
	ApplyConfig(ac map[string]interface{})
	BuildBeacon() OperatorBeacon
}

type AgentConfig struct {
	Name           string
	AESKey         string
	Range          string
	Contact        string
	Address        string
	Useragent      string
	Sleep          int
	KillSleep      int
	CommandJitter  int
	CommandTimeout int
	Pid            int
	Proxy          string
	Debug          bool
	Executing      map[string]Instruction
}

type OperatorBeacon struct {
	Name      string
	Target    string
	Hostname  string
	Location  string
	Platform  string
	Executors []string
	Range     string
	Sleep     int
	Pwd       string
	Executing string
	Links     []Instruction
}

type Instruction struct {
	ID       string `json:"ID"`
	Executor string `json:"Executor"`
	Payload  string `json:"Payload"`
	Request  string `json:"Request"`
	Response string
	Status   int
	Pid      int
}

func (c *AgentConfig) BuildExecutingHash() string {
	if count := len(c.Executing); count > 0 {
		ids := make([]string, count)
		for id := range c.Executing {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		h := md5.New()
		for _, s := range ids {
			io.WriteString(h, s)
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	return ""
}

func (c *AgentConfig) StartInstruction(instruction Instruction) bool {
	if _, ex := c.Executing[instruction.ID]; ex {
		return false
	}
	c.Executing[instruction.ID] = instruction
	return true
}

func (c *AgentConfig) StartInstructions(instructions []Instruction) (ret []Instruction) {
	for _, i := range instructions {
		if c.StartInstruction(i) {
			ret = append(ret, i)
		}
	}
	return
}

func (c *AgentConfig) EndInstruction(instruction Instruction) {
	delete(c.Executing, instruction.ID)
}
