package main

import (
	"fmt"
	"image-primitive/primitive"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

		ext := filepath.Ext(header.Filename)[1:]
		out, err := primitive.Transform(file, ext, 50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		outFile, err := tempfile(ext)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer outFile.Close()
		io.Copy(outFile, out)
		redirectURL := fmt.Sprintf("%s", outFile.Name())
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
	fileServer := http.FileServer(http.Dir("./img/"))
	mux.Handle("/img/", http.StripPrefix("/img", fileServer))
	log.Fatal(http.ListenAndServe(":3000", mux))
}

func tempfile(ext string) (*os.File, error) {
	in, err := ioutil.TempFile("./img/", "out_")
	if err != nil {
		// TODO: improve this error handling, perhaps retry?
		return nil, err
	}
	defer os.Remove(in.Name())
	return os.Create(fmt.Sprintf("%s.%s", in.Name(), ext))
}
