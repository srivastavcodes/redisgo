package main

import (
	"bufio"
	"fmt"
	"io"
)

// Writer writes RESP values to an io.Writer.
type Writer struct {
	writer *bufio.Writer
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: bufio.NewWriter(w)}
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
