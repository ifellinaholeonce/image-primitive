package main

import (
	"image-primitive/primitive"
	"io"
	"os"
)

func main() {
	image, err := os.Open("tmp/starry_night.jpg")
	if err != nil {
		panic(err)
	}
	defer image.Close()
	out, err := primitive.Transform(image, 50)
	if err != nil {
		panic(err)
	}
	os.Remove("out.jpg")
	outFile, err := os.Create("out.jpg")
	if err != nil {
		panic(err)
	}
	io.Copy(outFile, out)
}
