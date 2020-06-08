package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func main() {
	out, err := primitive("tmp/starry_night.jpg", "out.png", 100, triangle)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}

type PrimitiveMode int

const (
	combo PrimitiveMode = iota
	triangle
	rect
	ellipse
	circle
	rotatedrect
	beziers
	rotatedellipse
	polygon
)

func primitive(inputFile, outputFile string, numShapes int, mode PrimitiveMode) (string, error) {
	argStr := fmt.Sprintf("-i %s -o %s -n %d -m %d", inputFile, outputFile, numShapes, mode)
	cmd := exec.Command("primitive", strings.Fields(argStr)...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}
