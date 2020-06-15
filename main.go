package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"primitive"
	"strconv"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/joho/godotenv"
)

type genOpts struct {
	N        int
	M        primitive.Mode
	FilePath string
}

func main() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		for {
			<-ticker.C
			files, err := ioutil.ReadDir("./img/")
			for _, file := range files {
				ageLimit := time.Now().Add(time.Duration(-10 * time.Minute))
				if file.ModTime().Before(ageLimit) {
					err := os.Remove("./img/" + file.Name())
					if err != nil {
						panic(err)
					}
					fmt.Printf("Removed %v \n", file.Name())
				}
			}
			if err != nil {
				panic(err)
			}
		}
	}()
	err := godotenv.Load()
	if err != nil {
		// Don't panic when no env, for heroku.
		fmt.Println("Error loading .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
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
		onDisk, err := tempfile(ext)
		if err != nil {
			panic(err)
		}
		defer onDisk.Close()
		_, err = io.Copy(onDisk, file)
		if err != nil {
			panic(err)
		}
		http.Redirect(w, r, "/transform/"+filepath.Base(onDisk.Name()), http.StatusFound)
	})

	mux.HandleFunc("/transform/", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("./img/" + filepath.Base(r.URL.Path))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		defer f.Close()
		ext := filepath.Ext(f.Name())[1:]
		modeStr := r.FormValue("mode")
		if modeStr == "" {
			opts := []genOpts{
				{N: 50, M: primitive.ModeBeziers},
				{N: 50, M: primitive.ModeCombo},
				{N: 50, M: primitive.ModeRotatedRect},
				{N: 50, M: primitive.ModeRotatedEllipse},
			}
			for i := range opts {
				file, err := tempfile(ext)
				if err != nil {
					panic(err)
				}
				opts[i].FilePath = file.Name()
			}
			go renderModeChoices(w, r, f.Name(), ext, opts...)
			html := buildHTML()
			tpl := template.Must(template.New("").Parse(html))
			var fingerprints []string
			for _, opt := range opts {
				base := filepath.Base(opt.FilePath)
				fingerprints = append(fingerprints, base[0:len(base)-(len(ext)+1)])
			}
			err = tpl.Execute(w, fingerprints)
			if err != nil {
				panic(err)
			}
			return
		}
		mode, err := strconv.Atoi(modeStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		nStr := r.FormValue("n")
		if nStr == "" {
			renderNumShapesChoices(w, r, f, ext, primitive.Mode(mode))
			return
		}
		numShapes, err := strconv.Atoi(nStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = numShapes
		http.Redirect(w, r, "/img/"+filepath.Base(f.Name()), http.StatusFound)
	})

	fileServer := http.FileServer(http.Dir("./img/"))
	mux.Handle("/img/", http.StripPrefix("/img", fileServer))
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func tempfile(ext string) (*os.File, error) {
	in, err := ioutil.TempFile("./img/", "")
	if err != nil {
		// TODO: improve this error handling, perhaps retry?
		return nil, err
	}
	defer os.Remove(in.Name())
	return os.Create(fmt.Sprintf("%s.%s", in.Name(), ext))
}

func renderNumShapesChoices(w http.ResponseWriter, r *http.Request, f io.ReadSeeker, ext string, mode primitive.Mode) {
	opts := []genOpts{
		{N: 10, M: mode},
		// {N: 20, M: mode},
		// {N: 40, M: mode},
		// {N: 80, M: mode},
	}
	imgs, err := genImages(f, ext, opts...)
	if err != nil {
		panic(err)
	}

	html := `<html><body>
		{{range .}}
			<a href="/transform/{{.Name}}?mode={{.Mode}}&n={{.NumShapes}}">
				<img style="width: 20%;" src="/img/{{.Name}}">
			</a>
		{{end}}
		</body></html>`
	type dataStruct struct {
		Name      string
		Mode      primitive.Mode
		NumShapes int
	}
	var data []dataStruct
	for i, img := range imgs {
		data = append(data, dataStruct{
			Name:      filepath.Base(img),
			Mode:      opts[i].M,
			NumShapes: opts[i].N,
		})
	}
	tpl := template.Must(template.New("").Parse(html))
	err = tpl.Execute(w, data)
	if err != nil {
		panic(err)
	}
}

func renderModeChoices(w http.ResponseWriter, r *http.Request, imgPath string, ext string, opts ...genOpts) {
	f, err := os.Open(imgPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	imgs, err := genImages(f, ext, opts...)
	if err != nil {
		panic(err)
	}
	_ = imgs
	// TODO: Return this HTML async
	// html := `<html><body>
	// 	{{range .}}
	// 		<a href="/transform/{{.Name}}?mode={{.Mode}}">
	// 			<img style="width: 20%;" src="/img/{{.Name}}">
	// 		</a>
	// 	{{end}}
	// 	</body></html>`
	// type dataStruct struct {
	// 	Name string
	// 	Mode primitive.Mode
	// }
	// var data []dataStruct
	// for i, img := range imgs {
	// 	data = append(data, dataStruct{
	// 		Name: filepath.Base(img),
	// 		Mode: opts[i].M,
	// 	})
	// }
	// tpl := template.Must(template.New("").Parse(html))
	// err = tpl.Execute(w, data)
	// if err != nil {
	// 	panic(err)
	// }
}

func genImages(rs io.ReadSeeker, ext string, opts ...genOpts) ([]string, error) {
	var ret []string
	for _, opt := range opts {
		rs.Seek(0, 0)
		f, err := genImage(rs, ext, opt.N, opt.M, opt.FilePath)
		if err != nil {
			return nil, err
		}
		ret = append(ret, f)
	}
	return ret, nil
}

func genImage(r io.Reader, ext string, numShapes int, mode primitive.Mode, outFilePath string) (string, error) {
	out, err := primitive.Transform(r, ext, numShapes, primitive.WithMode(mode))
	if err != nil {
		return "", err
	}
	dir, fName := filepath.Split(outFilePath)
	outFile, err := os.Create(dir + "out_" + fName)
	if err != nil {
		return "", err
	}
	defer outFile.Close()
	_, err = io.Copy(outFile, out)
	if err != nil {
		panic(err)
	}
	return outFile.Name(), nil
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

func buildHTML() string {
	html := `<html><body>
			<link href="https://stackpath.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css" rel="stylesheet" integrity="sha384-wvfXpqpZZVQGK6TAh5PVlGOfQNHSoD2xbE+QkPxCAFlNEevoEH3Sl0sibVcOQVnN" crossorigin="anonymous">
			<script type="text/javascript">
				async function sleep(ms) {
					return new Promise(resolve => setTimeout(resolve, ms));
				}
			</script>
			<script>
				function fetchRetry(url, delay, tries) {
						function onError(err){
								triesLeft = tries - 1;
								if(!triesLeft){
										throw err;
								}
								return sleep(delay).then(() => fetchRetry(url, delay, triesLeft));
						}
						return fetch(url).then((res) => {
						if (res.ok) {
								return url
							}
							throw new Error("HTTP status " + res.status)
						})
						.catch(onError);
				}
			</script>
			<p>The names are going to be:</p>
			<ul>
				{{range.}}
					<li id="{{.}}"><i class="fa fa-spinner fa-spin" style="font-size:24px"></i></li>
				{{end}}
			</ul>
			<script type="text/javascript">
				async function getImg() {
					await sleep(10000)
					{{range.}}
						console.log("loaded script for {{.}}")
						console.log("fetching {{.}}")
						await fetchRetry("http://localhost:3000/img/out_{{.}}.jpg", 2000, 10).then((val) => {
							el = document.getElementById("{{.}}")
							var img = document.createElement('img')
							img.src = "http://localhost:3000/img/out_{{.}}.jpg"
							img.style.width = "20%;"
							img.id = "{{.}}"
							el.parentNode.replaceChild(img, el)
						})
					{{end}}
				}
				getImg()
			</script>
			</body></html>`
	return html
}
