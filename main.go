package main

import (
	"fmt"
	"image-primitive/primitive"
	"io"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<html></body>
		<form action="/upload" method="post" enctype="multipart/form-data">
			<input name="image" type="file">
			<button type="submit">Upload Image</button>
		</form>
		</body></html>`
		fmt.Fprint(w, html)
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		defer file.Close()
		// TODO: Handle file extensions
		ext := filepath.Ext(header.Filename)
		_ = ext
		out, err := primitive.Transform(file, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "image/png")
		io.Copy(w, out)
	})
	log.Fatal(http.ListenAndServe(":3000", mux))
}
