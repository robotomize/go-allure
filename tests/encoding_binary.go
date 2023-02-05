package tests

import (
	"encoding/binary"
	"fmt"
)

type simpleStruct struct {
	Name     string
	LastName string
}

func Marshal(s simpleStruct) ([]byte, error) {
	nameSize := len(s.Name)
	lastNameSize := len(s.LastName)
	buf := make([]byte, 0, 4+nameSize+4+lastNameSize)

	buf = binary.BigEndian.AppendUint32(buf, uint32(nameSize))
	buf = append(buf, []byte(s.Name)...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(lastNameSize))
	buf = append(buf, []byte(s.LastName)...)

	return buf, nil
}

func Unmarshal(b []byte, s *simpleStruct) error {
	// @TODO need check nil ptr
	if len(b) < 4 {
		return fmt.Errorf("format invalid")
	}

	nameLen := binary.BigEndian.Uint32(b[:4])
	if len(b) < 4+int(nameLen) {
		return fmt.Errorf("name length invalid")
	}
	s.Name = string(b[4 : 4+nameLen])
	lastNameLen := binary.BigEndian.Uint32(b[4+nameLen : 4+nameLen+4])
	if len(b) < 4+int(nameLen)+4+int(lastNameLen) {
		return fmt.Errorf("last name length invalid")
	}

	s.LastName = string(b[4+nameLen+4 : 4+nameLen+4+lastNameLen])

	return nil
}
