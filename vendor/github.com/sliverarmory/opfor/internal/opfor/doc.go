// Package opfor implements an embeddable, pure-Go Sleep and Aggressor Script
// runtime.
//
// Compile turns a named source unit into an immutable Program. Runtime executes
// programs, retains loaded declarations, dispatches events and other bindings,
// and exposes explicit Host and ObjectHost boundaries for behavior supplied by
// an embedding application. Standard output, standard error, and standard
// input default to their process streams and can be replaced with options to
// New. Arguments passed to Load, Execute, and Eval populate Sleep's @ARGV;
// they are not top-level @_ or positional subroutine arguments.
//
// Host handles unresolved application functions and ObjectHost handles
// Java-style object syntax. Opaque values implementing Iterator or
// MutableIterator participate in Sleep iterator consumers and foreach removal.
// Importer-defined ordinary, filter, and predicate environment keywords are
// registered with WithEnvironment and compiled through Runtime.Compile;
// Binding exposes the corresponding selector, filter, predicate, and nested
// popup/menu composition metadata.
// Known Cobalt-owned function families use typed provider interfaces carrying
// resolved Values, guarded callbacks, and runtime/script/source provenance;
// WithFunction provides exact overrides and Host remains the generic fallback.
// These ordinary in-process boundaries do not require serialization.
// These extension points are not a security sandbox: portable Sleep filesystem
// and process functions perform local effects unless an importer overrides them
// or constrains the process externally.
//
// OPFOR is an independent compatibility implementation. It does not include a
// Cobalt Strike client or Team Server implementation.
package opfor
