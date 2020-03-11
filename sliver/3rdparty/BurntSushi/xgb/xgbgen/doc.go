/*
xgbgen constructs Go source files from xproto XML description files. xgbgen
accomplishes the same task as the Python code generator for XCB and xpyb.

Usage:
	xgbgen [flags] some-protocol.xml

The flags are:
	--proto-path path
		The path to a directory containing xproto XML description files.
		This is only necessary when 'some-protocol.xml' imports other
		protocol files.
	--gofmt=true
		When false, the outputted Go code will not be gofmt'd. And it won't
		be very pretty at all. This is typically useful if there are syntax
		errors that need to be debugged in code generation. gofmt will hiccup;
		this will allow you to see the raw code.

How it works

xgbgen works by parsing the input XML file using Go's encoding/xml package.
The majority of this work is done in xml.go and xml_fields.go, where the
appropriate types are declared.

Due to the nature of the XML in the protocol description files, the types
required to parse the XML are not well suited to reasoning about code
generation. Because of this, all data parsed in the XML types is translated
into more reasonable types. This translation is done in translation.go,
and is mainly grunt work. (The only interesting tidbits are the translation
of XML names to Go source names, and connecting fields with their
appropriate types.)

The organization of these types is greatly
inspired by the description of the XML found here:
http://cgit.freedesktop.org/xcb/proto/tree/doc/xml-xcb.txt

These types come with a lot of supporting methods to make their use in
code generation easier. They can be found in expression.go, field.go,
protocol.go, request_reply.go and type.go. Of particular interest are
expression evaluation and size calculation (in bytes).

These types also come with supporting methods that convert their
representation into Go source code. I've quartered such methods in
go.go, go_error.go, go_event.go, go_list.go, go_request_reply.go,
go_single_field.go, go_struct.go and go_union.go. The idea is to keep
as much of the Go specific code generation in one area as possible. Namely,
while not *all* Go related code is found in the 'go*.go' files, *most*
of it is. (If there's any interest in using xgbgen for other languages,
I'd be happy to try and make xgbgen a little more friendly in this regard.
I did, however, design xgbgen with this in mind, so it shouldn't involve
anything as serious as a re-design.)

Why

I wrote xgbgen because I found the existing code generator that was written in
Python to be unwieldy. In particular, static and strong typing greatly helped
me reason better about the code generation task.

What does not work

The core X protocol should be completely working. As far as I know, most
extensions should work too, although I've only tested (and not much) the
Xinerama and RandR extensions.

XKB does not work. I don't have any real plans of working on this unless there
is demand and I have some test cases to work with. (i.e., even if I could get
something generated for XKB, I don't have the inclination to understand it
enough to verify that it works.) XKB poses several extremely difficult
problems that XCB also has trouble with. More info on that can be found at
http://cgit.freedesktop.org/xcb/libxcb/tree/doc/xkb_issues and
http://cgit.freedesktop.org/xcb/libxcb/tree/doc/xkb_internals.

*/
package main
