package opfor

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	aggressorUtilityChunkSize        = 32 * 1024
	aggressorIPRangeMaximumAddresses = 1 << 16
)

// PortableUtilityArgumentError identifies an argument that cannot be handled
// by one of OPFOR's pure-Go Aggressor utility functions. These errors describe
// OPFOR's independently specified safe boundaries; they do not claim the
// undocumented failure behavior of a licensed Cobalt Strike runtime.
type PortableUtilityArgumentError struct {
	Function string
	Position int
	Reason   string
}

func (err *PortableUtilityArgumentError) Error() string {
	if err == nil {
		return "opfor: invalid portable utility argument"
	}
	name := builtinName(err.Function)
	if name == "" {
		name = "portable utility"
	}
	if err.Position > 0 {
		return fmt.Sprintf("&%s: argument %d %s", name, err.Position, err.Reason)
	}
	return fmt.Sprintf("&%s: %s", name, err.Reason)
}

// aggressorPortableUtilityFunctions contains the documented Aggressor helpers
// that have no Cobalt client, Team Server, or data-model dependency. String
// byte operations use the low eight bits of each Java UTF-16 code unit, the
// same portable boundary used by OPFOR's pack, base64, and binary I/O helpers.
func (r *Runtime) aggressorPortableUtilityFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"format_size":        builtinAggressorFormatSize,
		"gunzip":             builtinAggressorGunzip,
		"gzip":               builtinAggressorGzip,
		"iprange":            builtinAggressorIPRange,
		"powershell_command": r.builtinAggressorPowerShellCommand,
		"script_resource":    builtinAggressorScriptResource,
		"str_chunk":          builtinAggressorStrChunk,
		"str_decode":         builtinAggressorStrDecode,
		"str_encode":         builtinAggressorStrEncode,
		"str_xor":            builtinAggressorStrXOR,
		"transform":          builtinAggressorTransform,
	}
}

type aggressorIPv4Span struct {
	start uint32
	count uint64
}

func builtinAggressorIPRange(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	description := invocation.Arg(0).String()
	parts := strings.Split(description, ",")
	spans := make([]aggressorIPv4Span, 0, len(parts))
	var total uint64
	for _, part := range parts {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		part = strings.TrimSpace(part)
		if part == "" {
			return Null(), aggressorIPRangeArgumentError(invocation, "must not contain an empty range item")
		}
		span, err := parseAggressorIPv4Span(part)
		if err != nil {
			return Null(), aggressorIPRangeArgumentError(invocation, err.Error())
		}
		if span.count > aggressorIPRangeMaximumAddresses-total {
			return Null(), aggressorIPRangeArgumentError(invocation,
				fmt.Sprintf("expands to more than %d IPv4 addresses", aggressorIPRangeMaximumAddresses))
		}
		total += span.count
		spans = append(spans, span)
	}

	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	if err := reserveCollectionEntryAmount(invocation.Runtime, total); err != nil {
		return Null(), err
	}
	addresses := make([]Value, 0, int(total))
	position := 0
	for _, span := range spans {
		for offset := uint64(0); offset < span.count; offset++ {
			if position%aggressorUtilityChunkSize == 0 {
				if err := ctx.Err(); err != nil {
					return Null(), err
				}
			}
			addresses = append(addresses, String(aggressorUint32IPv4(span.start+uint32(offset)).String()))
			position++
		}
	}
	return ArrayValue(NewArray(addresses...)), nil
}

func aggressorIPRangeArgumentError(invocation Invocation, reason string) error {
	return &PortableUtilityArgumentError{
		Function: invocation.Name,
		Position: 1,
		Reason:   reason,
	}
}

func parseAggressorIPv4Span(item string) (aggressorIPv4Span, error) {
	if strings.Contains(item, "/") {
		prefix, err := netip.ParsePrefix(item)
		if err != nil || !prefix.Addr().Is4() {
			return aggressorIPv4Span{}, fmt.Errorf("contains invalid IPv4 prefix %q", item)
		}
		prefix = prefix.Masked()
		return aggressorIPv4Span{
			start: aggressorIPv4Uint32(prefix.Addr()),
			count: uint64(1) << uint(32-prefix.Bits()),
		}, nil
	}

	if strings.Contains(item, "-") {
		if strings.Count(item, "-") != 1 {
			return aggressorIPv4Span{}, fmt.Errorf("contains invalid IPv4 range %q", item)
		}
		bounds := strings.SplitN(item, "-", 2)
		startText := strings.TrimSpace(bounds[0])
		endText := strings.TrimSpace(bounds[1])
		start, ok := parseAggressorIPv4(startText)
		if !ok {
			return aggressorIPv4Span{}, fmt.Errorf("contains invalid IPv4 range start %q", startText)
		}

		end, ok := parseAggressorIPv4(endText)
		if !ok && !strings.Contains(endText, ".") {
			lastOctet, err := strconv.ParseUint(endText, 10, 8)
			if err == nil {
				bytes := start.As4()
				bytes[3] = byte(lastOctet)
				end = netip.AddrFrom4(bytes)
				ok = true
			}
		}
		if !ok {
			return aggressorIPv4Span{}, fmt.Errorf("contains invalid IPv4 range end %q", endText)
		}

		startValue := aggressorIPv4Uint32(start)
		endValue := aggressorIPv4Uint32(end)
		if endValue <= startValue {
			return aggressorIPv4Span{}, fmt.Errorf("contains non-ascending IPv4 range %q", item)
		}
		return aggressorIPv4Span{start: startValue, count: uint64(endValue - startValue)}, nil
	}

	address, ok := parseAggressorIPv4(item)
	if !ok {
		return aggressorIPv4Span{}, fmt.Errorf("contains invalid IPv4 address %q", item)
	}
	return aggressorIPv4Span{start: aggressorIPv4Uint32(address), count: 1}, nil
}

func parseAggressorIPv4(value string) (netip.Addr, bool) {
	address, err := netip.ParseAddr(value)
	return address, err == nil && address.Is4()
}

func aggressorIPv4Uint32(address netip.Addr) uint32 {
	bytes := address.As4()
	return uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3])
}

func aggressorUint32IPv4(value uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)})
}

func builtinAggressorScriptResource(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	resource := filepath.FromSlash(invocation.Arg(0).String())
	if filepath.IsAbs(resource) {
		return String(filepath.Clean(resource)), nil
	}
	base, err := aggressorScriptResourceBase(invocation)
	if err != nil {
		return Null(), err
	}
	return String(filepath.Clean(filepath.Join(base, resource))), nil
}

func aggressorScriptResourceBase(invocation Invocation) (string, error) {
	var sourceName string
	if invocation.Runtime != nil {
		invocation.Runtime.mu.RLock()
		script := invocation.Runtime.scripts[invocation.Script]
		invocation.Runtime.mu.RUnlock()
		if script != nil && script.program != nil {
			sourceName = script.program.source.Name
		}
	}
	if sourceName == "" {
		sourceName = invocation.Span.Source
	}
	if sourceName != "" && sourceName != "STDIN" &&
		!(strings.HasPrefix(sourceName, "<") && strings.HasSuffix(sourceName, ">")) {
		identity := sourceIdentity(sourceName)
		if separator := strings.Index(identity, "!/"); separator >= 0 {
			identity = identity[:separator]
		}
		return filepath.Dir(filepath.FromSlash(identity)), nil
	}
	if invocation.Runtime != nil && invocation.Runtime.defaultFileResolver != nil {
		return invocation.Runtime.defaultFileResolver.BaseDirectory(), nil
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("&%s: determine script resource directory: %w", builtinName(invocation.Name), err)
	}
	return workingDirectory, nil
}

func builtinAggressorFormatSize(ctx context.Context, invocation Invocation) (Value, error) {
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	size := sleepFloat64(invocation.Arg(0))
	if math.IsNaN(size) || math.IsInf(size, 0) {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 1,
			Reason:   "must be a finite number",
		}
	}

	// The official contract guarantees the human-readable result 1kb for
	// 1024, but does not specify rounding or larger-unit edges. OPFOR uses a
	// deterministic binary scale and at most two fractional decimal places.
	units := [...]string{"b", "kb", "mb", "gb", "tb", "pb", "eb"}
	unit := 0
	for math.Abs(size) >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	size = math.Round(size*100) / 100
	if size == 0 {
		// Normalize negative zero so it does not leak into user-facing output.
		size = 0
	}
	return String(strconv.FormatFloat(size, 'f', -1, 64) + units[unit]), nil
}

func builtinAggressorGzip(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	input := sleepStringLowBytes(invocation.Arg(0))
	var output bytes.Buffer
	writer := newRuntimeOutputWriter(runtimeOutputAccountFor(ctx, invocation.Runtime), &output)
	compressor := gzip.NewWriter(writer)
	for position := 0; position < len(input); {
		if err := ctx.Err(); err != nil {
			_ = compressor.Close()
			return Null(), err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(input) {
			end = len(input)
		}
		if _, err := compressor.Write(input[position:end]); err != nil {
			_ = compressor.Close()
			return Null(), fmt.Errorf("&%s: compress input: %w", builtinName(invocation.Name), err)
		}
		position = end
	}
	if err := ctx.Err(); err != nil {
		_ = compressor.Close()
		return Null(), err
	}
	if err := compressor.Close(); err != nil {
		return Null(), fmt.Errorf("&%s: finish compressed input: %w", builtinName(invocation.Name), err)
	}
	return BinaryString(output.Bytes()), nil
}

func builtinAggressorGunzip(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	compressed := sleepStringLowBytes(invocation.Arg(0))
	decompressor, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return Null(), fmt.Errorf("&%s: open compressed input: %w", builtinName(invocation.Name), err)
	}
	defer decompressor.Close()

	var output bytes.Buffer
	buffer := make([]byte, aggressorUtilityChunkSize)
	for {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		amount, readErr := decompressor.Read(buffer)
		if amount != 0 {
			// Charge expanded bytes before retaining them. The shared Runtime
			// account makes concurrent gunzip calls and ScriptLoader children
			// compete for one monotonic family budget.
			if err := invocation.Runtime.reserveResource(resourceDecompressedBytes, uint64(amount)); err != nil {
				return Null(), err
			}
			_, _ = output.Write(buffer[:amount])
		}
		switch readErr {
		case nil:
			continue
		case io.EOF:
			return BinaryString(output.Bytes()), nil
		default:
			return Null(), fmt.Errorf("&%s: decompress input: %w", builtinName(invocation.Name), readErr)
		}
	}
}

func builtinAggressorStrChunk(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	maximum := sleepInt64(invocation.Arg(1))
	if maximum <= 0 {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason:   "maximum chunk size must be greater than zero",
		}
	}
	value := sleepStringCoercion(invocation.Arg(0))
	length := sleepStringLength(value)
	count := int64(length) / maximum
	if int64(length)%maximum != 0 {
		count++
	}
	if err := reserveCollectionEntryAmount(invocation.Runtime, uint64(count)); err != nil {
		return Null(), err
	}
	chunks := make([]Value, 0)
	for start := 0; start < length; {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		end64 := int64(start) + maximum
		end := length
		if end64 < int64(length) {
			end = int(end64)
		}
		chunks = append(chunks, sleepStringValueSlice(value, start, end))
		start = end
	}
	return ArrayValue(NewArray(chunks...)), nil
}

func builtinAggressorStrEncode(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	charset, err := lookupAggressorTextCharset(invocation.Arg(1).String())
	if err != nil {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason:   err.Error(),
		}
	}
	units := sleepStringUnits(invocation.Arg(0))
	var encoder sleepTextEncoder
	encoder.reset(charset)
	output := make([]byte, 0, len(units)*2)
	for position := 0; position < len(units); {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(units) {
			end = len(units)
		}
		output = append(output, encoder.encodeUnits(units[position:end], false)...)
		position = end
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	output = append(output, encoder.finish()...)
	return BinaryString(output), nil
}

func builtinAggressorStrDecode(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	charset, err := lookupAggressorTextCharset(invocation.Arg(1).String())
	if err != nil {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason:   err.Error(),
		}
	}
	input := sleepStringLowBytes(invocation.Arg(0))
	var decoder sleepTextDecoder
	decoder.reset(charset)
	units := make([]uint16, 0, len(input))
	for position := 0; position < len(input); {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(input) {
			end = len(input)
		}
		units = append(units, decoder.decode(input[position:end], false)...)
		position = end
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	units = append(units, decoder.decode(nil, true)...)
	return sleepStringValueFromUnits(units, nil), nil
}

func builtinAggressorStrXOR(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	input := sleepStringLowBytes(invocation.Arg(0))
	key := sleepStringLowBytes(invocation.Arg(1))
	if len(key) == 0 {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason:   "XOR key must not be empty",
		}
	}
	output := make([]byte, len(input))
	for position := range input {
		if position%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return Null(), err
			}
		}
		output[position] = input[position] ^ key[position%len(key)]
	}
	return BinaryString(output), nil
}

func lookupAggressorTextCharset(name string) (sleepTextCharset, error) {
	// The official examples spell this family UTF16-LE. Accept that spelling
	// in addition to the Java-standard aliases shared with BasicIO.
	switch strings.ToLower(name) {
	case "utf16-le":
		name = "utf-16le"
	case "utf16-be":
		name = "utf-16be"
	}
	return sleepLookupTextCharset(name)
}
