package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type ValueType string

const (
	String  ValueType = "+"
	Array   ValueType = "*"
	Bulk    ValueType = "$"
	Integer ValueType = ":"
	Null    ValueType = ""
	Error   ValueType = "-"
)

type Value struct {
	Type  ValueType
	Bulk  string
	Str   string
	Int   int64
	Err   string
	Array []Value
}

// readLine reads a line from the reader, trimming the newline character.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

// readArray reads an array from the reader.
func (v *Value) readArray(r *bufio.Reader) error {
	line, err := readLine(r)
	if err != nil {
		return err
	}
	if len(line) == 0 || line[0] != '*' {
		return fmt.Errorf("expected array line, got %s", line)
	}
	arrLen, err := strconv.Atoi(line[1:])
	if err != nil {
		return err
	}
	v.Array = make([]Value, arrLen)

	for i := 0; i < arrLen; i++ {
		bulk, err := v.readBulk(r)
		if err != nil {
			return err
		}
		v.Array[i] = bulk
	}
	v.Type = Array
	return nil
}

// readBulk reads a bulk string from the reader.
func (v *Value) readBulk(r *bufio.Reader) (val Value, err error) {
	line, err := readLine(r)
	if err != nil {
		return val, err
	}
	if len(line) == 0 || line[0] != '$' {
		return val, fmt.Errorf("expected bulk line, got %s", line)
	}
	bulkLen, err := strconv.Atoi(line[1:])
	if err != nil {
		return val, err
	}
	if bulkLen == -1 {
		val.Type = Null
		return val, nil
	}
	buf := make([]byte, bulkLen+2)

	if _, err = io.ReadFull(r, buf); err != nil {
		return val, err
	}
	val.Type, val.Bulk = Bulk, string(buf[:bulkLen])

	return val, nil
}
