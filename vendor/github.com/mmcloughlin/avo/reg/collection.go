package reg

// Collection represents a collection of virtual registers. This is primarily
// useful for allocating virtual registers with distinct IDs.
type Collection struct {
	vid map[Kind]VID
}

// NewCollection builds an empty register collection.
func NewCollection() *Collection {
	return &Collection{
		vid: map[Kind]VID{},
	}
}

// VirtualRegister allocates and returns a new virtual register of the given kind and width.
func (c *Collection) VirtualRegister(k Kind, w Width) Virtual {
	vid := c.vid[k]
	c.vid[k]++
	return NewVirtual(vid, k, w)
}

// GP8 allocates and returns a general-purpose 8-bit register.
func (c *Collection) GP8() GPVirtual { return c.GP(B8) }

// GP16 allocates and returns a general-purpose 16-bit register.
func (c *Collection) GP16() GPVirtual { return c.GP(B16) }

// GP32 allocates and returns a general-purpose 32-bit register.
func (c *Collection) GP32() GPVirtual { return c.GP(B32) }

// GP64 allocates and returns a general-purpose 64-bit register.
func (c *Collection) GP64() GPVirtual { return c.GP(B64) }

// GP allocates and returns a general-purpose register of the given width.
func (c *Collection) GP(w Width) GPVirtual { return newgpv(c.VirtualRegister(KindGP, w)) }

// XMM allocates and returns a 128-bit vector register.
func (c *Collection) XMM() VecVirtual { return c.Vec(B128) }

// YMM allocates and returns a 256-bit vector register.
func (c *Collection) YMM() VecVirtual { return c.Vec(B256) }

// ZMM allocates and returns a 512-bit vector register.
func (c *Collection) ZMM() VecVirtual { return c.Vec(B512) }

// Vec allocates and returns a vector register of the given width.
func (c *Collection) Vec(w Width) VecVirtual { return newvecv(c.VirtualRegister(KindVector, w)) }
