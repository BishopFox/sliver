package models

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"database/sql"
	"database/sql/driver"
	"fmt"
	uuid "uuid"
)

const uuidSize = 16

// UUID stores a Go standard library UUID in database models. The standard
// library type intentionally does not implement database/sql's Scanner and
// driver.Valuer interfaces, so this named type supplies that persistence
// boundary while keeping the canonical string representation used by Sliver's
// existing databases.
type UUID uuid.UUID

var (
	_ sql.Scanner   = (*UUID)(nil)
	_ driver.Valuer = UUID{}
)

// NewUUID returns a random version 4 UUID.
func NewUUID() UUID {
	return UUID(uuid.NewV4())
}

// NilUUID returns the all-zero UUID.
func NilUUID() UUID {
	return UUID(uuid.Nil())
}

// UUIDFrom converts a standard library UUID to its database representation.
func UUIDFrom(value uuid.UUID) UUID {
	return UUID(value)
}

// ParseUUID parses a textual UUID into its database representation.
func ParseUUID(value string) (UUID, error) {
	// Go 1.27's uuid.Parse accepts compact UUIDs, braced canonical UUIDs,
	// and canonical URNs. Preserve the two compact wrapped forms that the
	// previous gofrs/uuid parser also accepted.
	switch len(value) {
	case 34:
		if value[0] == '{' && value[len(value)-1] == '}' {
			value = value[1 : len(value)-1]
		}
	case 41:
		if value[:9] == "urn:uuid:" {
			value = value[9:]
		}
	}
	parsed, err := uuid.Parse(value)
	return UUID(parsed), err
}

// ParseUUIDOrNil parses a textual UUID and returns NilUUID on failure.
func ParseUUIDOrNil(value string) UUID {
	parsed, err := ParseUUID(value)
	if err != nil {
		return NilUUID()
	}
	return parsed
}

// UUIDFromBytes converts exactly 16 raw UUID bytes into a database UUID.
func UUIDFromBytes(value []byte) (UUID, error) {
	if len(value) != uuidSize {
		return NilUUID(), fmt.Errorf("uuid: UUID must be exactly 16 bytes long, got %d bytes", len(value))
	}
	var parsed UUID
	copy(parsed[:], value)
	return parsed, nil
}

// Standard returns the equivalent standard library UUID.
func (u UUID) Standard() uuid.UUID {
	return uuid.UUID(u)
}

// String returns the canonical lowercase UUID representation.
func (u UUID) String() string {
	return u.Standard().String()
}

// AppendText implements encoding.TextAppender.
func (u UUID) AppendText(dst []byte) ([]byte, error) {
	return u.Standard().AppendText(dst)
}

// MarshalText implements encoding.TextMarshaler.
func (u UUID) MarshalText() ([]byte, error) {
	return u.Standard().MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UUID) UnmarshalText(value []byte) error {
	parsed, err := ParseUUID(string(value))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// Value implements driver.Valuer using the canonical string representation.
func (u UUID) Value() (driver.Value, error) {
	return u.String(), nil
}

// Scan implements sql.Scanner for UUID strings and raw 16-byte values.
func (u *UUID) Scan(src any) error {
	switch value := src.(type) {
	case UUID:
		*u = value
		return nil
	case uuid.UUID:
		*u = UUID(value)
		return nil
	case []byte:
		if len(value) == uuidSize {
			copy(u[:], value)
			return nil
		}
		return u.UnmarshalText(value)
	case string:
		return u.UnmarshalText([]byte(value))
	default:
		return fmt.Errorf("uuid: cannot convert %T to UUID", src)
	}
}
