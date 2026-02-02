package output

import (
	"fmt"
	"io"
	"os"
	"reflect"
)

// ChangeType represents the type of change detected in watch mode
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "ADDED"
	ChangeTypeModified ChangeType = "MODIFIED"
	ChangeTypeDeleted  ChangeType = "DELETED"
)

// Change represents a detected change in watch mode
type Change struct {
	Type     ChangeType
	Resource interface{}
	Field    string
	OldValue interface{}
	NewValue interface{}
}

// WatchPrinterInterface is implemented by printers that support watch mode
type WatchPrinterInterface interface {
	Printer
	PrintChanges(changes []Change) error
}

type WatchPrinter struct {
	basePrinter Printer
	writer      io.Writer
	colorize    bool
}

func NewWatchPrinter(basePrinter Printer) *WatchPrinter {
	return &WatchPrinter{
		basePrinter: basePrinter,
		writer:      os.Stdout,
		colorize:    true,
	}
}

func NewWatchPrinterWithWriter(basePrinter Printer, writer io.Writer, colorize bool) *WatchPrinter {
	return &WatchPrinter{
		basePrinter: basePrinter,
		writer:      writer,
		colorize:    colorize,
	}
}

func (p *WatchPrinter) Print(data interface{}) error {
	return p.basePrinter.Print(data)
}

func (p *WatchPrinter) PrintList(data interface{}) error {
	return p.basePrinter.PrintList(data)
}

func (p *WatchPrinter) PrintChanges(changes []Change) error {
	for _, change := range changes {
		var prefix string
		var color string

		switch change.Type {
		case ChangeTypeAdded:
			prefix = "+"
			color = Green
		case ChangeTypeModified:
			prefix = "~"
			color = Yellow
		case ChangeTypeDeleted:
			prefix = "-"
			color = Red
		default:
			prefix = " "
			color = ""
		}

		if err := p.printWithPrefix(change.Resource, prefix, color); err != nil {
			return err
		}
	}
	return nil
}

func (p *WatchPrinter) printWithPrefix(resource interface{}, prefix string, color string) error {
	// For table output, we need to format as a single row without headers
	if tablePrinter, ok := p.basePrinter.(*TablePrinter); ok {
		return p.printTableRow(resource, prefix, color, tablePrinter)
	}

	// For other formats, print the prefix and the resource
	if p.colorize && color != "" {
		fmt.Fprintf(p.writer, "%s%s%s ", color, prefix, Reset)
	} else {
		fmt.Fprintf(p.writer, "%s ", prefix)
	}

	return p.basePrinter.Print(resource)
}

func (p *WatchPrinter) printTableRow(resource interface{}, prefix string, color string, tablePrinter *TablePrinter) error {
	// Format the resource as a table row string
	v := reflect.ValueOf(resource)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		// Fallback for non-struct types
		if p.colorize && color != "" {
			fmt.Fprintf(p.writer, "%s%s%s %v\n", color, prefix, Reset, resource)
		} else {
			fmt.Fprintf(p.writer, "%s %v\n", prefix, resource)
		}
		return nil
	}

	// Get field values using reflection (same logic as TablePrinter)
	t := v.Type()
	fields := getTableFields(t, tablePrinter.wide)

	var values []string
	for _, f := range fields {
		value := getFieldByPath(v, f.indices)
		values = append(values, formatValue(value))
	}

	// Print prefix and row values with proper spacing
	if p.colorize && color != "" {
		fmt.Fprintf(p.writer, "%s%s%s ", color, prefix, Reset)
	} else {
		fmt.Fprintf(p.writer, "%s ", prefix)
	}

	// Print values with kubectl-style spacing (3 spaces between columns)
	for i, val := range values {
		if i > 0 {
			fmt.Fprintf(p.writer, "   ")
		}
		fmt.Fprintf(p.writer, "%s", val)
	}
	fmt.Fprintln(p.writer)

	return nil
}
