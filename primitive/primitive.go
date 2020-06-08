package primitive

import (
	"fmt"
	"os/exec"
	"strings"
)

// Mode defines shapes for image transformation
type Mode int

// Modes used by the Primitive packages
const (
	ModeCombo Mode = iota
	ModeTriangle
	ModeRect
	ModeEllipse
	ModeCircle
	ModeRotatedRect
	ModeBeziers
	ModeRotatedEllipse
	ModePolygon
)

func Primitive(inputFile, outputFile string, numShapes int, mode Mode) (string, error) {
	argStr := fmt.Sprintf("-i %s -o %s -n %d -m %d", inputFile, outputFile, numShapes, mode)
	cmd := exec.Command("primitive", strings.Fields(argStr)...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}
