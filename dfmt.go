package main

import (
	"fmt"
	"os"
	"strings"

	mowcli "github.com/jawher/mow.cli"
)

// metadata for internal use, must be var for linker
var (
	Version string = "0.0.1"
	tag     string = "dev"
	appName string = "dfmt"
)

// CLI option names and descriptions
const (
	prettyPrintOptName        = "pretty-print p"
	inputTypeOptName          = "input-format i"
	outputTypeOptName         = "output-format o"
	fieldDelimOptName         = "field-delimiter F"
	recordDelimOptName        = "record-delimiter R"
	verboseOptName            = "verbose v"
	stringTo64bfNumberOptName = "parse-to-finite-64b-number"
	inputName                 = "INPUT"
	outputName                = "OUTPUT"

	inputTypeDesc  = "input format"
	outputTypeDesc = "output format"
	inputDesc      = "input file (or stdin if not provided)"
	outputDesc     = "output file (or stdout if not provided)"
	verboseDesc    = "produce slightly more verbose output"
)

var (
	prettyPrintDesc = "[" +
		formatNameJSON + "," + formatNameYAML + "," + formatNameTOML +
		"] produce humand-friendly output"
	fieldDelimDesc         = "[" + formatNameCSF + "] field delimiter"
	recordDelimDesc        = "[" + formatNameCSF + "] record delimiter"
	stringToJSONNumberDesc = "[" + formatNameCSF + "," + formatNameINI + "] " +
		`attempts to convert strings to JSON
Numbers (64-bit signed or finite double floats)`
)

var (
	inputFormats = []string{
		formatNameJSON, formatNameYAML, formatNameTOML,
		formatNameStrings, formatNameNTStr, formatNameCSF,
		formatNameINI,
		autoFormat}
	outputFormats = []string{
		formatNameJSON, formatNameYAML, formatNameTOML,
		autoFormat}
	inputFormatsList  = strings.Join(inputFormats, ", ")
	outputFormatsList = strings.Join(outputFormats, ", ")

	toolLongDescription = fmt.Sprintf(`Supported input format values: 
    %s
Supported output format values: 
    %s

Strings are EOL-separated strings, %s are null-terminated strings. 

%s represents ".ini" files with case-insensitive keys. Settings outside 
any section are added to a '_' section. This section is omitted if empty.

Character-separated fields (CSFs) can be imported by specifying the field
and record separators. Unlike many CSV parsers, this tool applies no special 
escaping or treatment for variable field counts. Special character names: 
NL (new line), CR (carriage return), LF (line feed), NUL (\x00), or 
TAB (tabulator).

The behaviour of CSFs configured without a field delimiter and with NL or NUL
is undefined. It may behave like lines or null-terminated strings but this
may change at any time and may not be consistent across subcommands. 

%s output of anything but maps and objects is added to a global key '_' 
as a key is required.

For %s and %s, 64-bit signed integer and finite float conversions are 
attempted if the '--%s' option is given, otherwise the 
string representation is kept (see README.md for details).
This may result in larger numbers being rounded to a 64-bit float
representation.

Arbitrarily large numbers are not currently supported. Suggestions
and code contributions for dealing with them across formats are welcome .`,
		inputFormatsList, outputFormatsList,
		formatNameNTStr,
		formatNameINI,
		formatNameTOML,
		formatNameINI, formatNameCSF, strings.Split(stringTo64bfNumberOptName, " ")[0])
)

// CLI option and argument values
var (
	prettyPrint        bool   = false
	inputType          string = autoFormat
	outputType         string = autoFormat
	stringToJSONNumber bool   = false
	fieldDelim         string = ","
	recordDelim        string = "NL"
	input              string = ""
	output             string = ""
	verbose            bool   = false
)

func main() {
	err := configureApp().Run(os.Args)
	if err != nil {
		exit(exitInputError+exitTransformError+exitOutputError+exitConfigurationError, err.Error())
	}
}

func configureApp() *mowcli.Cli {
	var app = mowcli.App(appName, "A data file multi-tool.")
	app.LongDesc = toolLongDescription

	app.Command("convert",
		"Converts data files.\n\n"+"Also see `"+appName+" --help` for details.",
		func(cmd *mowcli.Cmd) {
			cmd.BoolOptPtr(&prettyPrint, prettyPrintOptName, false, prettyPrintDesc)
			cmd.StringOptPtr(&inputType, inputTypeOptName, autoFormat, inputTypeDesc)
			cmd.StringOptPtr(&outputType, outputTypeOptName, autoFormat, outputTypeDesc)
			cmd.StringOptPtr(&fieldDelim, fieldDelimOptName, ",", fieldDelimDesc)
			cmd.StringOptPtr(&recordDelim, recordDelimOptName, "NL", recordDelimDesc)
			cmd.BoolOptPtr(&stringToJSONNumber, stringTo64bfNumberOptName, false, stringToJSONNumberDesc)
			cmd.StringArgPtr(&input, inputName, "", inputDesc)
			cmd.StringArgPtr(&output, outputName, "", outputDesc)

			cmd.Spec = "[OPTIONS] [INPUT] [OUTPUT]"

			cmd.Action = func() {
				inputFormat, transformer, outputFormat := configureFormats()
				err := ConvertFile(input, inputFormat, transformer, output, outputFormat)
				if err != nil {
					exit(exitTransformError, err.Error())
				}
			}
		})

	app.Command("remove-nulls",
		"Converts data files and removes 'null' entries.",
		func(cmd *mowcli.Cmd) {
			cmd.BoolOptPtr(&prettyPrint, prettyPrintOptName, false, prettyPrintDesc)
			cmd.StringOptPtr(&inputType, inputTypeOptName, autoFormat, inputTypeDesc)
			cmd.StringOptPtr(&outputType, outputTypeOptName, autoFormat, outputTypeDesc)
			cmd.StringOptPtr(&fieldDelim, fieldDelimOptName, ",", fieldDelimDesc)
			cmd.StringOptPtr(&recordDelim, recordDelimOptName, "NL", recordDelimDesc)
			cmd.BoolOptPtr(&stringToJSONNumber, stringTo64bfNumberOptName, false, stringToJSONNumberDesc)
			var (
				rmValues   = cmd.BoolOpt("values v", false, "remove key-value pairs whose value is null")
				rmElements = cmd.BoolOpt("elements e", false, "remove array elements that are null")
			)
			cmd.StringArgPtr(&input, inputName, "", inputDesc)
			cmd.StringArgPtr(&output, outputName, "", outputDesc)

			cmd.Spec = "[OPTIONS] [INPUT] [OUTPUT]"
			cmd.LongDesc = "If none of the removal options are provided, a simple format conversion is performed."

			cmd.Action = func() {
				inputFormat, transformer, outputFormat := configureFormats()
				transformer = NewMultiTransformer(transformer, NilRemovalTransformer{
					RemoveNilKeys:     false,
					RemoveNilValues:   *rmValues,
					RemoveNilElements: *rmElements,
				})
				err := ConvertFile(input, inputFormat, transformer, output, outputFormat)
				if err != nil {
					exit(exitTransformError, err.Error())
				}
			}
		})

	app.Command("version", "Prints the application version.", func(cmd *mowcli.Cmd) {
		cmd.BoolOptPtr(&verbose, verboseOptName, false, verboseDesc)
		cmd.Action = func() {
			if verbose {
				os.Stderr.WriteString(fmt.Sprintf("version %s (%s)\n", Version, tag))

			} else {
				os.Stderr.WriteString(fmt.Sprintf("version %s\n", Version))

			}
		}
	})

	return app
}

// Create formats and the default (import) transformer based
// on command line arguments.
func configureFormats() (InputFormat, Transformer, OutputFormat) {
	inputFormat, err := NewInputFormat(input, inputType, fieldDelim, recordDelim)
	if err != nil {
		exit(exitConfigurationError, err.Error())
	}
	outputFormat, err := NewOutputFormat(output, outputType, prettyPrint)
	if err != nil {
		exit(exitConfigurationError, err.Error())
	}
	var transformer Transformer = NopTransformer{}
	if stringToJSONNumber &&
		(inputFormat.Name() == formatNameINI || inputFormat.Name() == formatNameCSF) {
		transformer = NewConfigurableTransformer(StringToFiniteNumberParser, nil, nil, nil, nil)
	}
	return inputFormat, transformer, outputFormat
}
