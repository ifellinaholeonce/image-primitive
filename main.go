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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	if err != nil {
		log.Fatal("Error loading .env file")
	}
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
		writeToS3(outFile)
		redirectURL := fmt.Sprintf("%s", outFile.Name())
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})
	fileServer := http.FileServer(http.Dir("./img/"))
	mux.Handle("/img/", http.StripPrefix("/img", fileServer))
	log.Fatal(http.ListenAndServe(":"+port, mux))
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

func writeToS3(file *os.File) {
	s3Bucket := os.Getenv("AWS_S3_BUCKET")
	item := file.Name()

	// 2) Create an AWS session
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	})

	// 3) Create a new AWS S3 downloader
	uploader := s3manager.NewUploader(sess)

	uploadInput := &s3manager.UploadInput{
		Bucket: &s3Bucket,
		Key:    &item,
		Body:   file,
	}
	fmt.Println("bucket", s3Bucket)
	fmt.Println("key", item)
	// 4) Upload the item from the bucket.
	_, err := uploader.Upload(uploadInput)
	if err != nil {
		log.Fatalf("Unable to upload item %q, %v", item, err)
	}

	fmt.Println("Uploaded", item)
}
