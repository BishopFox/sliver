package regalloc

import (
	"fmt"
	"strings"
)

type regSet uint64

func (rs regSet) format(info *RegisterInfo) string { //nolint:unused
	var ret []string
	for i := 0; i < 64; i++ {
		if rs&(1<<uint(i)) != 0 {
			ret = append(ret, info.RealRegName(RealReg(i)))
		}
	}
	return strings.Join(ret, ", ")
}

func (rs regSet) has(r RealReg) bool {
	return rs&(1<<uint(r)) != 0
}

func (rs regSet) add(r RealReg) regSet {
	if r >= 64 {
		return rs
	}
	return rs | 1<<uint(r)
}

func (rs regSet) range_(f func(allocatedRealReg RealReg)) {
	for i := 0; i < 64; i++ {
		if rs&(1<<uint(i)) != 0 {
			f(RealReg(i))
		}
	}
}

type regInUseSet struct {
	set regSet
	vrs [64]VReg
}

func (rs *regInUseSet) reset() {
	rs.set = 0
	for i := range rs.vrs {
		rs.vrs[i] = VRegInvalid
	}
}

func (rs *regInUseSet) format(info *RegisterInfo) string { //nolint:unused
	var ret []string
	for i := 0; i < 64; i++ {
		if rs.set&(1<<uint(i)) != 0 {
			vr := rs.vrs[i]
			ret = append(ret, fmt.Sprintf("(%s->v%d)", info.RealRegName(RealReg(i)), vr.ID()))
		}
	}
	return strings.Join(ret, ", ")
}

func (rs *regInUseSet) has(r RealReg) bool {
	if r >= 64 {
		return false
	}
	return rs.set&(1<<uint(r)) != 0
}

func (rs *regInUseSet) get(r RealReg) VReg {
	if r >= 64 {
		return VRegInvalid
	}
	return rs.vrs[r]
}

func (rs *regInUseSet) remove(r RealReg) {
	if r >= 64 {
		return
	}
	rs.set &= ^(1 << uint(r))
	rs.vrs[r] = VRegInvalid
}

func (rs *regInUseSet) add(r RealReg, vr VReg) {
	if r >= 64 {
		return
	}
	rs.set |= 1 << uint(r)
	rs.vrs[r] = vr
}

func (rs *regInUseSet) range_(f func(allocatedRealReg RealReg, vr VReg)) {
	for i := 0; i < 64; i++ {
		if rs.set&(1<<uint(i)) != 0 {
			f(RealReg(i), rs.vrs[i])
		}
	}
}
