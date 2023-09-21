package chaos_tcp

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
)

func encode(msg *Message) (*bytes.Buffer, error) {

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(msg)
	if err != nil {
		return nil, err
	}

	return &buf, nil

}

func pack(msg *Message) (*bytes.Buffer, error) {
	// encode data
	buf, err := encode(msg)
	if err != nil {
		return nil, err
	}
	n := int64(buf.Len())
	if n > 4048 {
		return nil, fmt.Errorf("read error, data length > 4048, length=%d", n)
	}

	// write data length
	buffer := bytes.NewBuffer(nil)
	binary.Write(buffer, binary.BigEndian, n)

	// write data
	buffer.Write(buf.Bytes())
	return buffer, nil
}

func decode(buffer []byte, msg *Message) error {
	var buf = bytes.NewReader(buffer)

	dec := gob.NewDecoder(buf)

	err := dec.Decode(msg)

	if err != nil {
		return err
	}

	return nil
}

func read(r io.Reader, msg *Message) error {
	// read data length
	var length int64
	err := binary.Read(r, binary.BigEndian, &length)
	// check tcp error
	if /*err = tcpErrorCheck(err);*/ err != nil {
		return err
	}

	if length < 0 {
		return fmt.Errorf("read error, data length < 0, length=%d", length)
	}
	if length > 4048 {
		return fmt.Errorf("read error, data length > 4048, length=%d", length)
	}

	// rend data
	buffer := make([]byte, length)
	n, err := io.ReadFull(r, buffer)
	if err != nil {
		return fmt.Errorf("read error=%s", err.Error())
	}
	if int64(n) != length {
		return fmt.Errorf("read error")
	}

	// decode data
	return decode(buffer, msg)
}
