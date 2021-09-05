package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	mowcli "github.com/jawher/mow.cli"
)

const (
	exitNoError            int = 0
	exitInputError         int = 1
	exitOutputError        int = 2
	exitTransformError     int = 4
	exitConfigurationError int = 32
)

var (
	// Named delimiters and their corresonding code point.
	namedDelimiters map[string]string = map[string]string{
		"NL":   "",
		"TAB":  "\t",
		"CR":   "\r",
		"LF":   "\n",
		"NUL":  "\000",
		"NULL": "\000",
		"":     "",
	}

	// Default messages for certain exit codes.
	exitMessages map[int]string = map[int]string{
		exitNoError:            "",
		exitInputError:         "input error: could not read the data or unmarshal",
		exitOutputError:        "output error: could not marshal or write the data",
		exitTransformError:     "transform error: could not transform the data according to the arguments provided",
		exitConfigurationError: "configuration error",
	}
)

// Internal tool to convert a user-supplied named character to the actual character.
func normalizeDelim(label string) string {
	if len(label) == 1 {
		return label
	}
	delim, ok := namedDelimiters[label]
	if ok {
		return delim
	} else {
		exit(exitConfigurationError, "unknown delimiter '"+label+"'")
		return "" // unreachable
	}
}

// Read lines from a character stream (as a slice of bytes).
func readLines(data []byte) ([]string, error) {
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// Splits a character stream (as a byte slice) by the given separator and returns the delimited strings.
func readSeparatedStrings(data []byte, separator string) []string {
	return strings.Split(string(data), separator)
}

// Exits the application gracefully and with an error message.
func exit(code int, message string) {
	if message == "" {
		msg, found := exitMessages[code]
		if !found {
			message = "runtime error"
		} else {
			message = msg
		}
	}
	if message != "" {
		os.Stderr.WriteString(fmt.Sprintln(message))
	}
	mowcli.Exit(code)
}

// Determines if a slice of strings contains a given string ignoring case (strictly speaking under case-folding).
func containsFold(value string, slice []string) bool {
	for _, v := range slice {
		if strings.EqualFold(value, v) {
			return true
		}
	}
	return false
}

// Creates the actual indentation string of a given length.
// The indentation is 0 if pretty is false, otherwise of a
// length of count (if greater than 0) or a default indent,
// otherwise.
func createIndentString(pretty bool, count int) string {
	if pretty && count > 0 {
		return strings.Repeat(" ", count)
	} else if pretty {
		return "  "
	} else {
		return ""
	}
}

// A utility function to read, transform, and write data.
func ConvertStream(reader io.Reader, informat Unmarshaler, transformer Transformer, writer io.Writer, outformat Marshaler) error {
	data, err := informat.Unmarshal(reader)
	if err != nil {
		return err
	}
	var transformed interface{}
	if transformer != nil {
		transformed, err = transformer.Transform(data)
		if err != nil {
			return err
		}
	} else {
		transformed = data
	}
	return outformat.Marshal(transformed, writer)
}

// A utility function to read from a file, transform the format, and write the output.
// It treates empty file names and `-` indicate stdin/stdout.
func ConvertFile(infile string, informat Unmarshaler, transformer Transformer, outfile string, outformat Marshaler) error {
	var reader io.Reader
	if infile == "" || infile == "-" {
		reader = os.Stdin
	} else {
		var file, err = os.OpenFile(infile, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}

	var writer io.Writer
	if outfile == "" || outfile == "-" {
		writer = os.Stdout
	} else {
		var file, err = os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE, 0640)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = file
	}

	return ConvertStream(reader, informat, transformer, writer, outformat)
}

// Check if the value is nil (and doesn't panic if it is not a nil-able type).
// Don't check pointers transitively: may be a cycle.
func isNil(value interface{}) bool {
	vtype := reflect.TypeOf(value)
	if vtype == nil {
		return true
	}
	switch vtype.Kind() {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Chan, reflect.Func, reflect.Interface:
		if reflect.ValueOf(value).IsNil() {
			return true
		}
	}
	return false
}

// Convert all strings in a given slice to lower case.
func sliceToLower(sl []string) []string {
	var t []string = make([]string, len(sl))
	for n, s := range sl {
		t[n] = strings.ToLower(s)
	}
	return t
}
