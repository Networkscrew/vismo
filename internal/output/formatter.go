package output

import (
	"fmt"
	"io"
	"os"

	"github.com/Veinar/vismo/internal/managedfields"
	"github.com/Veinar/vismo/internal/source"
)

// Formatter renders vismo output sections.
type Formatter interface {
	RenderResource(yaml []byte)
	RenderSources(result *source.AnalysisResult)
	RenderFieldTrace(trace *managedfields.FieldTrace)
}

// TextFormatter renders human-readable text with optional ANSI colouring when stdout is a TTY.
type TextFormatter struct {
	w     io.Writer
	isTTY bool
}

// NewTextFormatter creates a TextFormatter writing to w.
func NewTextFormatter(w io.Writer) *TextFormatter {
	tty := false
	if f, ok := w.(*os.File); ok {
		fi, err := f.Stat()
		if err == nil {
			tty = (fi.Mode() & os.ModeCharDevice) != 0
		}
	}
	return &TextFormatter{w: w, isTTY: tty}
}

func (f *TextFormatter) header(title string) {
	if f.isTTY {
		fmt.Fprintf(f.w, "\033[1;36m─── %s ───\033[0m\n", title)
	} else {
		fmt.Fprintf(f.w, "--- %s ---\n", title)
	}
}

// RenderResource prints raw YAML of the resource.
func (f *TextFormatter) RenderResource(yaml []byte) {
	f.header("Resource YAML")
	f.w.Write(yaml) //nolint:errcheck
	fmt.Fprintln(f.w)
}

// RenderSources prints the detected source tools.
func (f *TextFormatter) RenderSources(result *source.AnalysisResult) {
	f.header("Detected Sources")
	if len(result.Sources) == 0 {
		fmt.Fprintln(f.w, "  (no sources detected)")
		return
	}
	for i, s := range result.Sources {
		fmt.Fprintf(f.w, "  %d. Type: %s\n", i+1, s.Type)
		if s.Name != "" {
			fmt.Fprintf(f.w, "     Name: %s\n", s.Name)
		}
		if s.Version != "" {
			fmt.Fprintf(f.w, "     Version: %s\n", s.Version)
		}
		for k, v := range s.Details {
			fmt.Fprintf(f.w, "     %s: %s\n", k, v)
		}
	}
	fmt.Fprintln(f.w)
}

// RenderFieldTrace prints the managedFields ownership trace.
func (f *TextFormatter) RenderFieldTrace(trace *managedfields.FieldTrace) {
	f.header("Field Ownership Trace")
	fmt.Fprintf(f.w, "Field: %s\n", trace.FieldPath)
	if trace.FinalValue != nil {
		fmt.Fprintf(f.w, "Final Value: %v\n", trace.FinalValue)
	} else {
		fmt.Fprintln(f.w, "Final Value: (not set)")
	}
	fmt.Fprintln(f.w)

	if trace.Note != "" {
		fmt.Fprintf(f.w, "Note: %s\n", trace.Note)
		return
	}

	fmt.Fprintln(f.w, "Ownership Trace (based on managedFields):")
	for i, o := range trace.Owners {
		fmt.Fprintf(f.w, "%d. Manager: '%s'\n", i+1, o.Manager)
		fmt.Fprintf(f.w, "   Operation: %s\n", o.Operation)
		if !o.Time.IsZero() {
			fmt.Fprintf(f.w, "   Timestamp: %s\n", o.Time.Format("2006-01-02T15:04:05Z"))
		}
		if o.APIVersion != "" {
			fmt.Fprintf(f.w, "   APIVersion: %s\n", o.APIVersion)
		}
		if i == 0 {
			fmt.Fprintln(f.w, "   (This is the final owner of the value)")
		}
		fmt.Fprintln(f.w)
	}
}
