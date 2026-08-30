package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorClientServiceSpec struct {
	operation    AggressorClientServiceOperation
	minimum      int
	maximum      int
	returnsValue bool
	callback     int
}

var aggressorClientServiceSpecs = map[string]aggressorClientServiceSpec{
	"getAggressorClient":   exactAggressorClientServiceSpec(AggressorClientServiceGetAggressorClient, 0, true),
	"get_cs_version":       exactAggressorClientServiceSpec(AggressorClientServiceGetCSVersion, 0, true),
	"mynick":               exactAggressorClientServiceSpec(AggressorClientServiceMyNick, 0, true),
	"users":                exactAggressorClientServiceSpec(AggressorClientServiceUsers, 0, true),
	"action":               exactAggressorClientServiceSpec(AggressorClientServiceAction, 1, false),
	"elog":                 exactAggressorClientServiceSpec(AggressorClientServiceEventLog, 1, false),
	"say":                  exactAggressorClientServiceSpec(AggressorClientServiceSay, 1, false),
	"privmsg":              exactAggressorClientServiceSpec(AggressorClientServicePrivateMessage, 2, false),
	"custom_event":         exactAggressorClientServiceSpec(AggressorClientServiceCustomEvent, 2, false),
	"custom_event_private": exactAggressorClientServiceSpec(AggressorClientServiceCustomEventPrivate, 3, false),
	"closeClient":          exactAggressorClientServiceSpec(AggressorClientServiceCloseClient, 0, false),
	"sync_download": {
		operation: AggressorClientServiceSyncDownload,
		minimum:   2,
		maximum:   3,
		callback:  2,
	},
}

func exactAggressorClientServiceSpec(
	operation AggressorClientServiceOperation,
	arguments int,
	returnsValue bool,
) aggressorClientServiceSpec {
	return aggressorClientServiceSpec{
		operation: operation, minimum: arguments, maximum: arguments,
		returnsValue: returnsValue, callback: -1,
	}
}

// aggressorClientServiceFunctions returns native wrappers around the
// importer-owned connected-client boundary. With no provider, every valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorClientServiceFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorClientServiceSpecs))
	for name := range aggressorClientServiceSpecs {
		functions[name] = r.aggressorClientService
	}
	return functions
}

func (r *Runtime) aggressorClientService(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorClientServiceSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor client service",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorClientServiceArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorClientServiceProvider
	if isNilInterface(provider) {
		// Preserve the raw Invocation on the compatibility path. Existing Hosts
		// continue to receive pass-by-name and ordinary bare-variable Cell
		// capabilities, and their result/error behavior is unchanged.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	request := AggressorClientServiceRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
		Arguments: values,
	}
	if spec.callback >= 0 && len(values) > spec.callback {
		request.Arguments = append([]Value(nil), values[:spec.callback]...)
		request.Arguments = append(request.Arguments, values[spec.callback+1:]...)
		if values[spec.callback].IsNull() {
			request.CallbackState = AggressorCallbackNull
		} else {
			callback, callbackErr := invocation.RetainCallback(values[spec.callback])
			if callbackErr != nil {
				if errors.Is(callbackErr, ErrInvalidCallable) {
					return Null(), fmt.Errorf("&%s: argument %d is not callable or $null: %w",
						builtinName(invocation.Name), spec.callback+1, callbackErr)
				}
				return Null(), callbackErr
			}
			request.CallbackState = AggressorCallbackCallable
			request.Callback = callback
		}
	}
	result, err := provider.HandleAggressorClientService(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if !spec.returnsValue {
		return Null(), nil
	}
	return result, nil
}

func requireAggressorClientServiceArguments(invocation Invocation, minimum, maximum int) error {
	count := len(invocation.Arguments)
	if count >= minimum && count <= maximum {
		return nil
	}
	if minimum == maximum {
		return requireExactAggressorClientArguments(invocation, minimum)
	}
	return fmt.Errorf("&%s: expected %d to %d argument(s), received %d",
		builtinName(invocation.Name), minimum, maximum, count)
}
