package opfor

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const aggressorPowerShellCommandHook = "POWERSHELL_COMMAND"

// builtinAggressorTransform implements three transformation algorithms named
// by the public Aggressor reference. powershell-base64 is anchored by the
// reference's exact "2 + 2" output. For hex and veil, lower-case hex digits and
// narrowing each Sleep UTF-16 code unit to its low byte are explicit provisional
// OPFOR policy: the public reference defines those transformations semantically
// but does not publish letter case or Java-string-to-byte edge behavior.
//
// The same reference describes array only as comma-separated byte values, VBA
// only as an array with newlines, and VBS only as an expression yielding a
// string. It does not specify whitespace, line wrapping, quoting, or chunking.
// OPFOR rejects those selectors explicitly rather than inventing output that
// would be presented as byte-for-byte Cobalt compatibility. transform_vbs
// remains importer-resolved for the same reason.
func builtinAggressorTransform(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	input := invocation.Arg(0)
	selector := invocation.Arg(1).String()
	switch selector {
	case "hex":
		data, err := aggressorTransformLowBytes(ctx, input)
		if err != nil {
			return Null(), err
		}
		return String(hex.EncodeToString(data)), nil
	case "powershell-base64":
		encoded, err := aggressorPowerShellBase64(ctx, input)
		if err != nil {
			return Null(), err
		}
		return String(encoded), nil
	case "veil":
		data, err := aggressorTransformLowBytes(ctx, input)
		if err != nil {
			return Null(), err
		}
		const digits = "0123456789abcdef"
		output := make([]byte, len(data)*4)
		for index, value := range data {
			if index%aggressorUtilityChunkSize == 0 {
				if err := ctx.Err(); err != nil {
					return Null(), err
				}
			}
			position := index * 4
			output[position] = '\\'
			output[position+1] = 'x'
			output[position+2] = digits[value>>4]
			output[position+3] = digits[value&0x0f]
		}
		return String(string(output)), nil
	case "array", "vba", "vbs":
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason: fmt.Sprintf(
				"transform selector %q has no complete output grammar in the public Aggressor reference and is intentionally unsupported",
				selector,
			),
		}
	default:
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Position: 2,
			Reason: fmt.Sprintf(
				"transform selector %q is unsupported; portable selectors are hex, powershell-base64, and veil",
				selector,
			),
		}
	}
}

// builtinAggressorPowerShellCommand first honors the newest active
// POWERSHELL_COMMAND hook. Without a hook it applies the two templates published
// as the default hook, using transform(..., "powershell-base64") semantics.
func (r *Runtime) builtinAggressorPowerShellCommand(ctx context.Context, invocation Invocation) (Value, error) {
	if r == nil {
		return Null(), fmt.Errorf("&%s: runtime is nil", builtinName(invocation.Name))
	}
	if err := requireAggressorCommandArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Resolve both pass-by-name arguments once. The hook and fallback must see
	// the same snapshot even if evaluating one argument mutates the other.
	values := invocation.Values()
	if hooks := r.Bindings(BindingHook, aggressorPowerShellCommandHook); len(hooks) != 0 {
		hook := hooks[len(hooks)-1]
		hookArguments := []Value{values[0], values[1]}
		hookContext, release, err := r.prepareBindingInvocation(ctx, hook, hookArguments)
		if err != nil {
			return Null(), err
		}
		defer release()
		result, invokeErr := hook.Callback.Invoke(hookContext, hookArguments...)
		if err := joinExecutionError(invokeErr, release); err != nil {
			return Null(), err
		}
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
		return result, nil
	}

	encoded, err := aggressorPowerShellBase64(ctx, values[0])
	if err != nil {
		return Null(), err
	}
	if values[1].Truth() {
		return String("powershell -nop -w hidden -encodedcommand " + encoded), nil
	}
	return String("powershell -nop -exec bypass -EncodedCommand " + encoded), nil
}

func aggressorTransformLowBytes(ctx context.Context, value Value) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	units := sleepStringUnits(value)
	output := make([]byte, len(units))
	for index, unit := range units {
		if index%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		output[index] = byte(unit)
	}
	return output, nil
}

func aggressorPowerShellBase64(ctx context.Context, value Value) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// PowerShell's -EncodedCommand consumes UTF-16LE. Encoding Sleep's actual
	// UTF-16 code units preserves non-BMP surrogate pairs. BinaryString octets are
	// represented as U+0000..U+00ff before encoding; that binary edge policy is
	// explicit and remains provisional until licensed-runtime differential data
	// is available.
	units := sleepStringUnits(value)
	encodedInput := make([]byte, len(units)*2)
	for index, unit := range units {
		if index%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		binary.LittleEndian.PutUint16(encodedInput[index*2:index*2+2], unit)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encodedInput), nil
}
