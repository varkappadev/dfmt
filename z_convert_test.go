package main

import (
	"strings"
	"testing"
)

const (
	test_yaml = `a: b
---
c: 1
---
---
d: "e f"
`
	test_json = `{
		"a": 1,
		"b": { "c": "d"},
		"e": null,
		"f": [0, 1, 2]
	}
`
	test_array_json = `["a", 1, null]
`
	test_csf = `a,b,c
1,2,3
`
)

var (
	jsonInputFormat, _      = NewInputFormat("", "json", "", "")
	yamlInputFormat, _      = NewInputFormat("", "YAML", "", "")
	csfCommaInputFormat, _  = NewInputFormat("", "csf", ",", "NL")
	csfCustomInputFormat, _ = NewInputFormat("", "csf", ",", "|")

	jsonOutputFormat, _ = NewOutputFormat("", "JSON", false)
	yamlOutputFormat, _ = NewOutputFormat("", "yaml", false)
	tomlOutputFormat, _ = NewOutputFormat("", "TOML", false)

	jsonIndentedOutputFormat, _ = NewOutputFormat("", "json", true)
	yamlIndentedOutputFormat    = YAMLFormat{PrettyPrint: true, Indentation: 8}
	tomlIndentedOutputFormat, _ = NewOutputFormat("", "toml", true)

	defaultTransformer    = NopTransformer{}
	jsonNumberTransformer = NewConfigurableTransformer(StringToFiniteNumberParser, nil, nil, nil, nil)
)

func processString(input string, iformat Unmarshaler, transformer Transformer, oformat Marshaler) (interface{}, string, error) {
	writer := &strings.Builder{}
	if transformer == nil {
		transformer = defaultTransformer
	}
	err := ConvertStream(strings.NewReader(input), iformat, transformer, writer, oformat)
	return writer.String(), writer.String(), err
}

func convertTransformAndTest(t *testing.T, input string, expected string, iformat Unmarshaler, transformer Transformer, oformat Marshaler) interface{} {
	data, output, err := processString(input, iformat, transformer, oformat)
	if err != nil {
		t.Error(err)
	}
	if data == nil {
		t.Error("empty input")
	}
	if output != expected {
		t.Errorf("failed to convert %s to %s correctly, found '%s' expected '%s'", iformat.(FileFormat).Name(), oformat.(FileFormat).Name(), output, expected)
	}
	return data

}

func convertAndTest(t *testing.T, input string, expected string, iformat Unmarshaler, oformat Marshaler) interface{} {
	return convertTransformAndTest(t, input, expected, iformat, nil, oformat)
}

func TestYamlToJson(t *testing.T) {
	convertAndTest(t, test_yaml, `[{"a":"b"},{"c":1},null,{"d":"e f"}]`, yamlInputFormat, jsonOutputFormat)
}

func TestYamlToToml(t *testing.T) {
	convertAndTest(t, `{"a": 1, "b": 0, "c": -0.3}`, `a = 1
b = 0
c = -0.3
`, yamlInputFormat, tomlOutputFormat)
}

func TestJsonToJson(t *testing.T) {
	convertAndTest(t, test_json, `{"a":1,"b":{"c":"d"},"e":null,"f":[0,1,2]}`, jsonInputFormat, jsonOutputFormat)
}

func TestJsonArrayToJson(t *testing.T) {
	convertAndTest(t, test_array_json, `["a",1,null]`,
		jsonInputFormat, jsonOutputFormat)
}

func TestCsfToJson(t *testing.T) {
	convertTransformAndTest(t, test_csf, `[["a","b","c"],[1,2,3]]`,
		csfCommaInputFormat, jsonNumberTransformer, jsonOutputFormat)
}

func TestCsfToYaml(t *testing.T) {
	convertTransformAndTest(t, test_csf, `- - a
  - b
  - c
- - 1
  - 2
  - 3
`, csfCommaInputFormat, jsonNumberTransformer, yamlOutputFormat)
}

func TestCsfToToml(t *testing.T) {
	convertTransformAndTest(t, test_csf, `_ = [["a", "b", "c"], [1, 2, 3]]
`, csfCommaInputFormat, jsonNumberTransformer, tomlOutputFormat)
}

func TestCsfParsing(t *testing.T) {
	convertTransformAndTest(t,
		`0,1,2.5|-Inf,NaN,inf|-0,1.99999,1E-05`,
		`[[0,1,2.5],["-Inf","NaN","inf"],[0,1.99999,0.00001]]`,
		csfCustomInputFormat, jsonNumberTransformer, jsonOutputFormat)
}

func TestStrings(t *testing.T) {
	format, _ := NewInputFormat("", "Strings", "", "")
	convertAndTest(t, "abc\ndef\n", `["abc","def"]`, format, jsonOutputFormat)
}

func TestNTStrings(t *testing.T) {
	format, _ := NewInputFormat("", "NTStr", "", "")
	convertAndTest(t, "a\000b\000", `["a","b"]`, format, jsonOutputFormat)
}

func TestTomlImport(t *testing.T) {
	format, _ := NewInputFormat("a.toml", "auto", "", "")
	convertAndTest(t, `[a]
b = 1
`, `{"a":{"b":1}}`, format, jsonOutputFormat)
}

func TestYamlExport(t *testing.T) {
	iformat, _ := NewInputFormat("a.yaml", "auto", "", "")
	oformat, _ := NewOutputFormat("b.yaml", "auto", false)
	convertAndTest(t, `a: "1"`, "a: \"1\"\n", iformat, oformat)
}

func TestMultiSectionIniImport(t *testing.T) {
	format, _ := NewInputFormat("b.ini", "auto", "", "")
	convertTransformAndTest(t, `[a]
b=1
c = hi
[d]
[e]
f =
g= 123456789012345678901234567890
`, `{"a":{"b":1,"c":"hi"},"d":{},"e":{"f":"","g":1.2345678901234568e+29}}`,
		format, jsonNumberTransformer, jsonOutputFormat)
}

func TestDefaultSectionIniImport(t *testing.T) {
	format, _ := NewInputFormat("b.ini", "auto", "", "")
	convertTransformAndTest(t, `a=3.14
[b]
c = -8
`, `{"_":{"a":3.14},"b":{"c":-8}}`, format, jsonNumberTransformer, jsonOutputFormat)
}

func TestStringsIndentedJson(t *testing.T) {
	format, _ := NewInputFormat("", "strings", "", "")
	convertAndTest(t, "abc\ndef\n",
		`[
  "abc",
  "def"
]`, format, jsonIndentedOutputFormat)
}

func TestStringsIndentedToml(t *testing.T) {
	format := jsonInputFormat
	input := `{"a": 1, "b": {"c": 2}}`
	convertAndTest(t, input,
		`a = 1.0

[b]
  c = 2.0
`, format, tomlIndentedOutputFormat)

	convertAndTest(t, input,
		`a = 1.0

[b]
c = 2.0
`, format, tomlOutputFormat)
}

func TestStringsIndentedYaml(t *testing.T) {
	format := jsonInputFormat
	input := `{"a": 1, "b": {"c": 2}}`
	convertAndTest(t, input,
		`a: 1
b:
        c: 2
`, format, yamlIndentedOutputFormat)
	convertAndTest(t, input,
		`a: 1
b:
  c: 2
`, format, yamlOutputFormat)
}
