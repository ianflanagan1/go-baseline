package baseline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
)

// defaultIndent is the indentation PinToolchain uses when devbox.json gives
// none to follow.
const defaultIndent = "  "

// devboxLayout is where PinToolchain finds the parts of devbox.json it
// edits, as byte offsets.
type devboxLayout struct {
	// open is just after the top-level object's opening brace.
	open int
	// lastMemberEnd is just after the top-level object's last value, or -1
	// when it has none.
	lastMemberEnd int
	// indent is the indentation of the top-level object's first key, or ""
	// when that key doesn't start its own line.
	indent string
	// env is the env object, or nil when there is none.
	env *envLayout
}

// envLayout is where PinToolchain finds the parts of devbox.json's env
// object it edits, as byte offsets.
type envLayout struct {
	open, close int
	// lastMemberEnd is just after the last value, or -1 when it has none.
	lastMemberEnd int
	// indent is the indentation of the first key, or "" when that key
	// doesn't start its own line.
	indent string
	// toolchainStart and toolchainEnd surround GOTOOLCHAIN's value; both are
	// -1 when env doesn't set it.
	toolchainStart, toolchainEnd int
}

// PinToolchain returns devbox.json with env GOTOOLCHAIN set to the Go release
// the baseline approves, adding it, and env, where they are missing. The rest
// of the file keeps its bytes. It returns an error when devbox.json isn't a
// JSON object, or env isn't an object of strings.
func PinToolchain(devbox []byte) ([]byte, error) {
	layout, err := scanDevbox(devbox)
	if err != nil {
		return nil, fmt.Errorf("devbox.json: %w", err)
	}
	value := strconv.Quote(toolchainEnv)
	member := `"GOTOOLCHAIN": ` + value
	unit := layout.indent
	if unit == "" {
		unit = defaultIndent
	}
	env := layout.env
	switch {
	case env == nil && layout.lastMemberEnd < 0:
		return splice(devbox, layout.open, layout.open, "\n"+unit+`"env": {`+"\n"+unit+unit+member+"\n"+unit+"}\n"), nil
	case env == nil:
		return splice(devbox, layout.lastMemberEnd, layout.lastMemberEnd, ",\n"+unit+`"env": {`+"\n"+unit+unit+member+"\n"+unit+"}"), nil
	case env.toolchainStart >= 0:
		return splice(devbox, env.toolchainStart, env.toolchainEnd, value), nil
	case env.lastMemberEnd < 0:
		return splice(devbox, env.open, env.close, "\n"+unit+unit+member+"\n"+unit), nil
	case env.indent == "":
		return splice(devbox, env.lastMemberEnd, env.lastMemberEnd, ", "+member), nil
	default:
		return splice(devbox, env.lastMemberEnd, env.lastMemberEnd, ",\n"+env.indent+member), nil
	}
}

// splice returns b with b[start:end] replaced by text.
func splice(b []byte, start, end int, text string) []byte {
	return slices.Concat(b[:start], []byte(text), b[end:])
}

func scanDevbox(devbox []byte) (devboxLayout, error) {
	dec := json.NewDecoder(bytes.NewReader(devbox))
	layout := devboxLayout{lastMemberEnd: -1}
	if err := expectDelim(dec, '{'); err != nil {
		return layout, err
	}
	layout.open = int(dec.InputOffset())
	members, err := scanMembers(dec, devbox, func(key string) error {
		if key != "env" || layout.env != nil {
			return decode(dec, &json.RawMessage{})
		}
		env, err := scanEnv(dec, devbox)
		layout.env = &env
		return err
	})
	if err != nil {
		return layout, err
	}
	layout.indent, layout.lastMemberEnd = members.indent, members.lastEnd
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return layout, errors.New("more follows the top-level object")
	}
	return layout, nil
}

func scanEnv(dec *json.Decoder, devbox []byte) (envLayout, error) {
	env := envLayout{lastMemberEnd: -1, toolchainStart: -1, toolchainEnd: -1}
	if err := expectDelim(dec, '{'); err != nil {
		return env, fmt.Errorf("env: %w", err)
	}
	env.open = int(dec.InputOffset())
	members, err := scanMembers(dec, devbox, func(key string) error {
		valueStart := skipSpaceAndColon(devbox, int(dec.InputOffset()))
		var value string
		if err := decode(dec, &value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		if key == "GOTOOLCHAIN" {
			env.toolchainStart, env.toolchainEnd = valueStart, int(dec.InputOffset())
		}
		return nil
	})
	if err != nil {
		return env, fmt.Errorf("env: %w", err)
	}
	env.indent, env.lastMemberEnd = members.indent, members.lastEnd
	env.close = int(dec.InputOffset()) - 1
	return env, nil
}

// members is what scanMembers finds out about an object's members.
type members struct {
	// indent is the indentation of the first key, or "" when that key
	// doesn't start its own line.
	indent string
	// lastEnd is just after the last value, or -1 when there is none.
	lastEnd int
}

// scanMembers reads an object's members, after its opening brace, up to and
// including its closing brace. For each one it reads the key and calls
// value, which reads the value.
func scanMembers(dec *json.Decoder, devbox []byte, value func(key string) error) (members, error) {
	found := members{lastEnd: -1}
	for dec.More() {
		keyStart := skipSpaceAndComma(devbox, int(dec.InputOffset()))
		key, err := readKey(dec)
		if err != nil {
			return found, err
		}
		if found.lastEnd < 0 {
			found.indent = indentAt(devbox, keyStart)
		}
		if err := value(key); err != nil {
			return found, err
		}
		found.lastEnd = int(dec.InputOffset())
	}
	return found, expectDelim(dec, '}')
}

func readKey(dec *json.Decoder) (string, error) {
	token, err := dec.Token()
	if err != nil {
		return "", fmt.Errorf("reading a key: %w", err)
	}
	key, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("want a key, got %v", token)
	}
	return key, nil
}

func decode(dec *json.Decoder, v any) error {
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("reading a value: %w", err)
	}
	return nil
}

func expectDelim(dec *json.Decoder, want json.Delim) error {
	token, err := dec.Token()
	if err != nil {
		return fmt.Errorf("want %v: %w", want, err)
	}
	if token != want {
		return fmt.Errorf("want %v, got %v", want, token)
	}
	return nil
}

// indentAt returns the spaces and tabs before offset i when they start its
// line, and "" otherwise.
func indentAt(b []byte, i int) string {
	start := i
	for start > 0 && (b[start-1] == ' ' || b[start-1] == '\t') {
		start--
	}
	if start == 0 || b[start-1] != '\n' {
		return ""
	}
	return string(b[start:i])
}

func skipSpaceAndComma(b []byte, from int) int {
	return skip(b, from, ',')
}

func skipSpaceAndColon(b []byte, from int) int {
	return skip(b, from, ':')
}

// skip returns the offset of the first byte at or after from that is neither
// JSON whitespace nor separator.
func skip(b []byte, from int, separator byte) int {
	i := from
	for i < len(b) && (isSpace(b[i]) || b[i] == separator) {
		i++
	}
	return i
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
