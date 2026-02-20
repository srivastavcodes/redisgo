package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ValueType represents the type of RESP value.
type ValueType string

const (
	String  ValueType = "+"
	Array   ValueType = "*"
	Bulk    ValueType = "$"
	Integer ValueType = ":"
	Null    ValueType = ""
	Error   ValueType = "-"
)

// Value represents a RESP value.
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

// Writer writes RESP values to an io.Writer.
type Writer struct {
	writer *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer: bufio.NewWriter(w),
	}
}

// Write writes the given val to the writer.
func (w *Writer) Write(val *Value) (err error) {
	switch val.Type {
	case String:
		_, err = fmt.Fprintf(w.writer, "+%s\r\n", val.Str)
	case Array:
		_, err = fmt.Fprintf(w.writer, "*%d\r\n", len(val.Array))
		if err != nil {
			return err
		}
		for i := range val.Array {
			if err = w.Write(&val.Array[i]); err != nil {
				return err
			}
		}
	case Bulk:
		_, err = fmt.Fprintf(w.writer, "$%d\r\n%s\r\n", len(val.Bulk), val.Bulk)
	case Integer:
		_, err = fmt.Fprintf(w.writer, ":%d\r\n", val.Int)
	case Null:
		_, err = fmt.Fprint(w.writer, "$-1\r\n")
	case Error:
		_, err = fmt.Fprintf(w.writer, "-%s\r\n", val.Err)
	default:
		return fmt.Errorf("invalid val type: %s", val.Type)
	}
	return err
}

// Flush flushes the writer to the underlying io.Writer.
func (w *Writer) Flush() error {
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("flushing writer failed: %w", err)
	}
	return nil
}
