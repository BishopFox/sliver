package ansi

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// MarginWriter is a Writer that applies indentation and padding around
// whatever you write to it.
type MarginWriter struct {
	w  io.Writer
	iw *IndentWriter
}

// NewMarginWriter returns a new MarginWriter.
func NewMarginWriter(ctx RenderContext, w io.Writer, rules StyleBlock) *MarginWriter {
	bs := ctx.blockStack

	var indentation uint
	var margin uint
	if rules.Indent != nil {
		indentation = *rules.Indent
	}
	if rules.Margin != nil {
		margin = *rules.Margin
	}

	pw := NewPaddingWriter(w, int(bs.Width(ctx)), func(_ io.Writer) { //nolint:gosec
		_, _ = renderText(w, rules.StylePrimitive, " ")
	})

	ic := " "
	if rules.IndentToken != nil {
		ic = *rules.IndentToken
	}
	iw := NewIndentWriter(pw, int(indentation+margin), func(_ io.Writer) { //nolint:gosec
		_, _ = renderText(w, bs.Parent().Style.StylePrimitive, ic)
	})

	return &MarginWriter{
		w:  lipgloss.NewWrapWriter(w),
		iw: iw,
	}
}

// Write writes to the margin writer and implements [io.Writer].
func (w *MarginWriter) Write(b []byte) (int, error) {
	n, err := w.iw.Write(b)
	if err != nil {
		return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
	}
	return n, nil
}

// Close closes the [MarginWriter].
func (w *MarginWriter) Close() error {
	var werr error
	if w, ok := w.w.(io.WriteCloser); ok {
		werr = w.Close()
	}

	return errors.Join(werr, w.iw.Close())
}

// PaddingFunc is a function that applies padding around whatever you write to it.
type PaddingFunc = func(w io.Writer)

// PaddingWriter is a writer that applies padding around whatever you write to
// it.
type PaddingWriter struct {
	Padding int
	PadFunc PaddingFunc
	w       *lipgloss.WrapWriter
	cache   bytes.Buffer
}

// NewPaddingWriter returns a new PaddingWriter.
func NewPaddingWriter(w io.Writer, padding int, padFunc PaddingFunc) *PaddingWriter {
	return &PaddingWriter{
		Padding: padding,
		PadFunc: padFunc,
		w:       lipgloss.NewWrapWriter(w),
	}
}

// Write writes to the padding writer.
func (w *PaddingWriter) Write(p []byte) (int, error) {
	// Use UTF-8 aware iteration to properly handle multi-byte characters (e.g., CJK)
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if r == '\n' { //nolint:nestif
			line := w.cache.String()
			linew := ansi.StringWidth(line)
			if w.Padding > 0 && linew < w.Padding {
				if w.PadFunc != nil {
					for n := 0; n < w.Padding-linew; n++ {
						w.PadFunc(w.w)
					}
				} else {
					_, err := io.WriteString(w.w, strings.Repeat(" ", w.Padding-linew))
					if err != nil {
						return 0, fmt.Errorf("glamour: error writing padding: %w", err)
					}
				}
			}
			w.cache.Reset()
		} else {
			// Write complete UTF-8 character bytes to cache
			w.cache.Write(p[i : i+size])
		}

		// Write complete UTF-8 character bytes to output
		_, err := w.w.Write(p[i : i+size])
		if err != nil {
			return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
		}
		i += size
	}

	return len(p), nil
}

// Close closes the [PaddingWriter].
func (w *PaddingWriter) Close() error {
	return w.w.Close() //nolint:wrapcheck
}

// IndentFunc is a function that applies indentation around whatever you write to
// it.
type IndentFunc = func(w io.Writer)

// IndentWriter is a writer that applies indentation around whatever you write to
// it.
type IndentWriter struct {
	Indent     int
	IndentFunc PaddingFunc
	w          io.Writer
	pw         *lipgloss.WrapWriter
	skipIndent bool
}

// NewIndentWriter returns a new IndentWriter.
func NewIndentWriter(w io.Writer, indent int, indentFunc IndentFunc) *IndentWriter {
	return &IndentWriter{
		Indent:     indent,
		IndentFunc: indentFunc,
		pw:         lipgloss.NewWrapWriter(w),
		w:          w,
	}
}

func (w *IndentWriter) resetPen() {
	style := w.pw.Style()
	link := w.pw.Link()
	if !style.IsZero() {
		_, _ = io.WriteString(w.w, ansi.ResetStyle)
	}
	if !link.IsZero() {
		_, _ = io.WriteString(w.w, ansi.ResetHyperlink())
	}
}

func (w *IndentWriter) restorePen() {
	style := w.pw.Style()
	link := w.pw.Link()
	if !style.IsZero() {
		_, _ = io.WriteString(w.w, style.String())
	}
	if !link.IsZero() {
		_, _ = io.WriteString(w.w, ansi.SetHyperlink(link.URL, link.Params))
	}
}

// Write writes to the indentation writer.
func (w *IndentWriter) Write(p []byte) (int, error) {
	// Use UTF-8 aware iteration to properly handle multi-byte characters (e.g., CJK)
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		if !w.skipIndent {
			w.resetPen()
			if w.IndentFunc != nil {
				for j := 0; j < w.Indent; j++ {
					w.IndentFunc(w.pw)
				}
			} else {
				_, err := io.WriteString(w.pw, strings.Repeat(" ", w.Indent))
				if err != nil {
					return 0, fmt.Errorf("glamour: error writing indentation: %w", err)
				}
			}

			w.skipIndent = true
			w.restorePen()
		}

		if r == '\n' {
			w.skipIndent = false
		}

		// Write complete UTF-8 character bytes to output
		_, err := w.pw.Write(p[i : i+size])
		if err != nil {
			return 0, fmt.Errorf("glamour: error writing bytes: %w", err)
		}
		i += size
	}

	return len(p), nil
}

// Close closes the [IndentWriter].
func (w *IndentWriter) Close() error {
	var werr error
	if w, ok := w.w.(io.WriteCloser); ok {
		werr = w.Close()
	}

	return errors.Join(werr, w.pw.Close())
}
