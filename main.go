package main

import (
	"fmt"
	"image-primitive/primitive"
)

func main() {
	out, err := primitive.Primitive("tmp/starry_night.jpg", "out.png", 100, primitive.ModeTriangle)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
