//go:build go1.27

package sqlite3

import (
	"errors"
	"fmt"
	"uuid"
)

// CreateModule registers a new virtual table module name.
// If create is nil, the virtual table is eponymous.
//
// https://sqlite.org/c3ref/create_module.html
func (c *Conn) CreateModule[T VTab](name string, create, connect VTabConstructor[T]) error {
	return CreateModule(c, name, create, connect)
}

// ExtensionInit loads an SQLite extension library.
//
// https://sqlite.org/loadext.html
func (c *Conn) ExtensionInit[Env any, Mod ExtensionLibrary](init func(env Env) Mod, info ExtensionInfo) error {
	return ExtensionInit(c, init, info)
}

func (c *Conn) uuid() error {
	const flags = DETERMINISTIC | INNOCUOUS
	return errors.Join(
		c.CreateFunction("uuid", 0, INNOCUOUS, uuidNew),
		c.CreateFunction("uuid", 1, INNOCUOUS, uuidNew),
		c.CreateFunction("uuid_str", 1, flags, uuidStr),
		c.CreateFunction("uuid_blob", 1, flags, uuidBlob))
}

func uuidNew(ctx Context, arg ...Value) {
	ver := 4
	if len(arg) > 0 {
		ver = arg[0].Int()
	}

	var u uuid.UUID
	switch ver {
	case 4:
		u = uuid.NewV4()
	case 7:
		u = uuid.NewV7()
	default:
		ctx.ResultError(fmt.Errorf("uuid: invalid version: %d", ver)) // notest
		return
	}

	ctx.ResultText(u.String())
}

func uuidFrom(arg Value) (u uuid.UUID, err error) {
	switch t := arg.Type(); t {
	case TEXT:
		err = u.UnmarshalText(arg.RawText())
		if err != nil {
			err = fmt.Errorf("uuid: %w", err)
		}

	case BLOB:
		blob := arg.RawBlob()
		if size := len(blob); size != len(u) {
			err = fmt.Errorf("uuid: invalid BLOB length: %d", size)
		} else {
			copy(u[:], blob)
		}

	default:
		err = fmt.Errorf("uuid: invalid type: %v", t)
	}
	return u, err
}

func uuidBlob(ctx Context, arg ...Value) {
	u, err := uuidFrom(arg[0])
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultBlob(u[:])
	}
}

func uuidStr(ctx Context, arg ...Value) {
	u, err := uuidFrom(arg[0])
	if err != nil {
		ctx.ResultError(err) // notest
	} else {
		ctx.ResultText(u.String())
	}
}
