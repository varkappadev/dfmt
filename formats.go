package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"reflect"
	"strings"

	toml "github.com/BurntSushi/toml"
	ini "github.com/go-ini/ini"
	yaml "gopkg.in/yaml.v3"
)

const (
	autoFormat = "auto"
)

var (
	formatNameJSON     string   = JSONFormat{}.Name()
	formatNameYAML     string   = YAMLFormat{}.Name()
	formatNameTOML     string   = TOMLFormat{}.Name()
	formatNameINI      string   = INIFormat{}.Name()
	formatNamesStrings []string = []string{"Lines", "Strings"}
	formatNameStrings  string   = formatNamesStrings[0]
	formatNamesNTStr   []string = []string{"NTStr", "NTStrings", "NTString", "NTS"}
	formatNameNTStr    string   = formatNamesNTStr[0]
	formatNameCSF      string   = "CSF"

	fidJSON     string   = strings.ToLower(formatNameJSON)
	fidYAML     string   = strings.ToLower(formatNameYAML)
	fidTOML     string   = strings.ToLower(formatNameTOML)
	fidINI      string   = strings.ToLower(formatNameINI)
	fidsStrings []string = sliceToLower(formatNamesStrings)
	fidsNTStr   []string = sliceToLower(formatNamesNTStr)
	fidCSF      string   = strings.ToLower(formatNameCSF)
)

type Unmarshaler interface {
	Unmarshal(reader io.Reader) (interface{}, error)
}

type Marshaler interface {
	Marshal(data interface{}, w io.Writer) error
}

type FileFormat interface {
	Name() string
	SupportedExtensions() []string
}

type InputFormat interface {
	FileFormat
	Unmarshaler
}

type OutputFormat interface {
	FileFormat
	Marshaler
}

type InputOutputFormat interface {
	FileFormat
	Marshaler
	Unmarshaler
}

type JSONFormat struct {
	PrettyPrint bool
	Indentation int
}

func (f JSONFormat) Name() string {
	return "JSON"
}

func (f JSONFormat) SupportedExtensions() []string {
	return []string{".json"}
}

func (f JSONFormat) Unmarshal(reader io.Reader) (interface{}, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var value interface{}
	err = json.Unmarshal(bytes, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (f JSONFormat) Marshal(data interface{}, w io.Writer) error {
	indent := createIndentString(f.PrettyPrint, f.Indentation)
	var (
		bytes []byte
		err   error
	)
	if f.PrettyPrint {
		bytes, err = json.MarshalIndent(data, "", indent)
	} else {
		bytes, err = json.Marshal(data)
	}
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

type YAMLFormat struct {
	PrettyPrint bool
	Indentation int
}

func (f YAMLFormat) Name() string {
	return "YAML"
}

func (f YAMLFormat) SupportedExtensions() []string {
	return []string{".yaml", ".yml"}
}

func (f YAMLFormat) Unmarshal(reader io.Reader) (interface{}, error) {
	decoder := yaml.NewDecoder(reader)
	var documents []interface{} = make([]interface{}, 0)
	for {
		var document interface{}
		err := decoder.Decode(&document)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			} else {
				return nil, err
			}
		}
		documents = append(documents, document)
	}
	if len(documents) == 1 {
		return documents[0], nil
	} else {
		return documents, nil
	}
}

func (f YAMLFormat) Marshal(data interface{}, w io.Writer) error {
	buffer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buffer)
	spaces := len(createIndentString(f.PrettyPrint, f.Indentation))
	if spaces < 2 {
		spaces = 2
	}
	encoder.SetIndent(spaces)

	err := encoder.Encode(data)
	if err != nil {
		return err
	}
	_, err = w.Write(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

type TOMLFormat struct {
	PrettyPrint bool
	Indentation int
	DefaultKey  string
}

func (f TOMLFormat) Name() string {
	return "TOML"
}

func (f TOMLFormat) SupportedExtensions() []string {
	return []string{".toml", ".tml"}
}

func (f TOMLFormat) Unmarshal(reader io.Reader) (interface{}, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	var value interface{}
	err = toml.Unmarshal(bytes, &value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (f TOMLFormat) Marshal(data interface{}, w io.Writer) error {
	buffer := &bytes.Buffer{}
	encoder := toml.NewEncoder(buffer)
	encoder.Indent = createIndentString(f.PrettyPrint, f.Indentation)

	var ndata interface{}
	switch reflect.ValueOf(data).Kind() {
	case reflect.Map, reflect.Struct:
		ndata = data
	default:
		ndata = map[string]interface{}{NonemptyDefaultKey(f.DefaultKey): data}
	}
	err := encoder.Encode(ndata)
	if err != nil {
		return err
	}
	_, err = w.Write(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

type TextFormat struct {
	RecordDelimiter string
	FieldDelimiter  string
}

func (f TextFormat) Name() string {
	if f.FieldDelimiter == "" {
		switch f.RecordDelimiter {
		case "":
			return formatNameStrings
		case "\000":
			return formatNameNTStr
		default:
			return formatNameCSF
		}
	} else {
		return formatNameCSF
	}
}

func (f TextFormat) SupportedExtensions() []string {
	return []string{}
}

func (f TextFormat) Unmarshal(reader io.Reader) (interface{}, error) {
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var records []string
	if f.RecordDelimiter == "" {
		records, err = readLines(bytes)
	} else {
		records = readSeparatedStrings(bytes, f.RecordDelimiter)
		if len(records) > 0 {
			if len(records[len(records)-1]) == 0 {
				records = records[0 : len(records)-1]
			}
		}
		err = nil
	}
	if err != nil {
		return nil, err
	}

	if f.FieldDelimiter == "" {
		return records, nil
	}

	var data []interface{} = make([]interface{}, 0, len(records))
	for _, record := range records {
		fields := readSeparatedStrings([]byte(record), f.FieldDelimiter)
		parsedFields := make([]interface{}, len(fields))
		for f, s := range fields {
			parsedFields[f] = s
		}
		data = append(data, parsedFields)
	}
	return data, nil
}

type INIFormat struct {
	CaseSensitive bool
	DefaultKey    string
}

func (f INIFormat) Name() string {
	return "INI"
}

func (f INIFormat) SupportedExtensions() []string {
	return []string{".ini"}
}

func (f INIFormat) Unmarshal(reader io.Reader) (interface{}, error) {
	var (
		file *ini.File
		err  error
	)
	if f.CaseSensitive {
		file, err = ini.Load(reader)
	} else {
		file, err = ini.InsensitiveLoad(reader)
	}
	if err != nil {
		return nil, err
	}
	var data map[string]map[string]interface{} = make(map[string]map[string]interface{})
	for _, section := range file.Sections() {
		name := section.Name()
		if name == "default" {
			name = NonemptyDefaultKey(f.DefaultKey)
			if len(section.KeysHash()) == 0 {
				continue
			}
		}
		data[name] = make(map[string]interface{})
		for k, v := range section.KeysHash() {
			data[name][k] = v
		}
	}
	if err != nil {
		return nil, err
	}

	return data, nil
}

func NewTextFormat(rdelim string, fdelim string) TextFormat {
	return TextFormat{
		RecordDelimiter: normalizeDelim(rdelim),
		FieldDelimiter:  normalizeDelim(fdelim),
	}
}

func NewFormat(fileName string, formatName string, fieldDelim string, recordDelim string, prettyPrint bool) (FileFormat, error) {
	var (
		jsonFormatConfig = JSONFormat{PrettyPrint: prettyPrint}
		yamlFormatConfig = YAMLFormat{PrettyPrint: prettyPrint}
		tomlFormatConfig = TOMLFormat{PrettyPrint: prettyPrint}
		iniFormatConfig  = INIFormat{CaseSensitive: false}
	)
	fid := strings.ToLower(formatName)
	switch fid {
	case fidJSON:
		return jsonFormatConfig, nil
	case fidYAML:
		return yamlFormatConfig, nil
	case fidTOML:
		return tomlFormatConfig, nil
	case fidCSF:
		return NewTextFormat(recordDelim, fieldDelim), nil
	case fidINI:
		return iniFormatConfig, nil
	default:
		if containsFold(fid, fidsStrings) {
			return NewTextFormat("NL", ""), nil
		} else if containsFold(fid, fidsNTStr) {
			return NewTextFormat("NUL", ""), nil
		}
	}

	if formatName != autoFormat {
		return nil, fmt.Errorf("unknown/unexpected format name '%s'", formatName)
	}

	ext := path.Ext(fileName)
	if containsFold(ext, JSONFormat{}.SupportedExtensions()) {
		return jsonFormatConfig, nil
	} else if containsFold(ext, YAMLFormat{}.SupportedExtensions()) {
		return yamlFormatConfig, nil
	} else if containsFold(ext, TOMLFormat{}.SupportedExtensions()) {
		return tomlFormatConfig, nil
	} else if containsFold(ext, INIFormat{}.SupportedExtensions()) {
		return iniFormatConfig, nil
	}

	return nil, fmt.Errorf("cannot determine format of file '%s'", fileName)
}

func NewInputFormat(fileName string, formatName string, fieldDelim string, recordDelim string) (InputFormat, error) {
	format, err := NewFormat(fileName, formatName, fieldDelim, recordDelim, false)
	if err != nil {
		return nil, err
	}
	informat, ok := format.(InputFormat)
	if ok {
		return informat, nil
	} else {
		return nil, fmt.Errorf("cannot configure input")
	}
}

func NewOutputFormat(fileName string, formatName string, prettyPrint bool) (OutputFormat, error) {
	format, err := NewFormat(fileName, formatName, "", "", prettyPrint)
	if err != nil {
		return nil, err
	}
	outformat, ok := format.(OutputFormat)
	if ok {
		return outformat, nil
	} else {
		return nil, fmt.Errorf("cannot configure output")
	}
}

func NonemptyDefaultKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return "_"
	} else {
		return key
	}
}
