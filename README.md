# Data File Multi-Tool

## Summary 

`dfmt` is a tool for converting data files between formats (JSON, YAML,
TOML, etc.).

Example usage: 

```console
dfmt convert in.json out.yaml
```

For command line options:

```console
dfmt --help
```

## Features

Currently, the following formats are supported

Format|Input|Output
------|-----|------
JSON|supported|supported
YAML|supported|supported
TOML|supported|supported
INI|supported|not supported
strings (by line or null-separated)|supported|not supported
character-separated fields (CSF)|supported|not supported

For INI and CSF files only, an attempt at converting strings consisting
of only finite numbers is made if the corresponding command line option 
is given. This may result in slightly different output such as missing 
surrounding spaces, rounding, etc. 

YAML multi-document files are supported but they are treated as an
array and will therefore be converted to a single document for all
formats, including to YAML itself.

## Thanks

Many thanks to the authors of the following libraries used in this
tool:

- https://github.com/BurntSushi/toml
- https://github.com/jawher/mow.cli
- https://github.com/go-ini/ini
- https://github.com/go-yaml/yaml

## Background and Limitations

I needed a utility for conversions – mostly to JSON – and this is the
second version of it, basically a port to golang. It does mostly what I
need. As such it may be totally useless/unsuitable for any other
purpose.

In particular, the tool is not meant to be optimized for speed or
memory consumption. It will hold the data in memory between read and
write.

*Additional limitations:*

- The CLI is not stable and it is not suitable for scripting at this 
point.

- Format-specific limitations on outputs apply and at the moment it is
not possible to configure things such as the default keys for TOML
output and INI input or case-(in)sensitivity of keys.

- If strings are converted to numbers, an attempt at converting them 
to signed 64-bit integers is made. If that fails, they are converted to 
floats with rounding. If this, in turn fails, they will be kept as
strings.

- Non-finite floating point numbers (`+Inf`, `-Inf`, `NaN`) are kept as
strings even in transformation cases that parse numbers in input.

Also check the [Issues
section](https://github.com/varkappadev/dfmt/issues) for reported bugs
and limitations.

## Issues and Contributions

Feel free to report bugs. Please include sample input to reproduce any
issues if possible.

Additional features will be limited to what I need personally. The
current plan is to include some transformations, mostly for data
clean-up.
