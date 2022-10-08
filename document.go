package shorthand

import (
	"bytes"
)

type OpKind int

const (
	OpSet OpKind = iota
	OpDelete
	OpSwap
)

type ParseOptions struct {
	// EnableFileInput turns on support for `@filename`` values which load from
	// files rather than being treated as string input.
	EnableFileInput bool

	// EnableObjectDetection will enable omitting the outer `{` and `}` for
	// objects, which can be useful for some applications such as command line
	// arguments.
	EnableObjectDetection bool

	// ForceStringKeys forces all map keys to be treated as strings, resulting
	// in all maps being of type `map[string]interface{}`. By default other types
	// are allowed, which will result in the use of `map[interface{}]interface{}`
	// for maps with non-string keys (`map[string]interface{}` is still the
	// default even when non-string keys are allowed).
	// If you know the output target will be JSON, you can enable this option
	// to efficiently create a result that will `json.Marshal(...)` safely.
	ForceStringKeys bool

	// DebugLogger sets a function to be used for printing out debug information.
	DebugLogger func(format string, a ...interface{})
}

type Operation struct {
	Kind  OpKind
	Path  string
	Value interface{}
}

type Document struct {
	Operations []Operation

	options    ParseOptions
	expression string
	pos        uint
	lastWidth  uint
	buf        bytes.Buffer
}

func NewDocument(options ParseOptions) *Document {
	return &Document{
		options: options,
	}
}

func (d *Document) String() string {
	// TODO: serialize to text format
	return ""
}

func (d *Document) Unmarshal(data []byte) {
	// TODO: load from JSON/CBOR representation
}

func (d *Document) Marshal() interface{} {
	ops := make([]interface{}, len(d.Operations))

	for i, op := range d.Operations {
		s := []interface{}{}
		if op.Kind != OpSet {
			s = append(s, op.Kind)
		}
		s = append(s, op.Path)
		if op.Value != nil {
			s = append(s, op.Value)
		}
		ops[i] = s
	}

	return ops
}

func (d *Document) Parse(input string) Error {
	d.expression = input
	d.pos = 0

	if d.options.EnableObjectDetection {
		// Try and determine if this is actually an object without the outer
		// `{` and `}` surrounding it. We re-use `parseProp`` for this as it
		// already handles things like quotes, escaping, etc.
		for {
			_, err := d.parseProp("", false)
			if err != nil {
				break
			}
			r := d.next()
			if r == ':' || r == '^' {
				// We have found an object! Wrap it and continue.
				d.expression = "{" + input + "}"
				if d.options.DebugLogger != nil {
					d.options.DebugLogger("Detected object, wrapping in { and }")
				}
			}
		}
		d.pos = 0
	}

	err := d.parseValue("", true, false)
	if err != nil {
		return err
	}
	d.skipWhitespace()
	if !d.expect(-1) {
		return d.error(1, "Expected EOF but found additional input")
	}
	return nil
}

func (d *Document) Apply(input interface{}) (interface{}, Error) {
	var err Error
	for _, op := range d.Operations {
		input, err = d.applyOp(input, op)
		if err != nil {
			return nil, err
		}
	}

	return input, nil
}
