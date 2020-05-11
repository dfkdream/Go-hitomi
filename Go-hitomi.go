package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	_ "golang.org/x/image/webp"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
)

//https://ba.hitomi.la/webp/e/49/7534d4bfe5d58bcfd1687352deb789f6f9d223a54b7f174fe5b431385216949e.webp

type GalleryInfo struct {
	LocalLang string      `json:"language_localname"`
	Lang      string      `json:"language"`
	Date      string      `json:"date"`
	Files     []ImageInfo `json:"files"`
	// Tags
	Id    string `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type ImageInfo struct {
	Width   uint   `json:"width"`
	Name    string `json:"name"`
	Height  uint   `json:"height"`
	Hash    string `json:"hash"`
	HasWebp int    `json:"haswebp"`
	HasAvif int    `json:"hasavif"`
}

type Result struct {
	Image   []byte
	ImgName string
	WK_ID   int
	IsWebp  bool
}

func GetImageNamesFromID(GalleryID string) []ImageInfo {
	_, resp, _ := fasthttp.Get(nil, "https://ltn.hitomi.la/galleries/"+GalleryID+".js")
	resp = bytes.Replace(resp, []byte("var galleryinfo = "), []byte(""), -1)
	var g GalleryInfo
	err := json.Unmarshal(resp, &g)
	if err != nil {
		log.Fatal(err)
	}
	return g.Files
}

func LnsCurrentDirectory() {
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))

	http.ListenAndServe(":80", nil)
}

func DownloadImage(url string, try int, signal chan<- string) []byte {
	for i := 0; i < try; i++ {
		if i != 0 {
			signal <- fmt.Sprintf("Redownloading %s: #%d/%d", url, i+1, try)
		}
		req := fasthttp.AcquireRequest()
		req.URI().Update(url)
		req.Header.SetMethod("GET")
		req.Header.Set("Referer", "https://hitomi.la")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:75.0) Gecko/20100101 Firefox/75.0")
		res := fasthttp.AcquireResponse()
		if err := Client.Do(req, res); err == nil && res.Header.StatusCode() == 200 && res.Header.ContentLength() > 0 {
			img := make([]byte, res.Header.ContentLength())
			copy(img, res.Body())
			fasthttp.ReleaseResponse(res)
			fasthttp.ReleaseRequest(req)
			return img
		} else {
			signal <- fmt.Sprintf("Download Error: %s: %d %v", url, res.Header.StatusCode(), err)
		}
		fasthttp.ReleaseResponse(res)
		fasthttp.ReleaseRequest(req)
	}
	signal <- "Download Failed: " + url
	return nil
}

func DownloadWorker(no int, rLimit int, signal chan<- string, ctrl <-chan struct{}, jobs <-chan ImageInfo, out chan<- Result) {
	for j := range jobs {
		select {
		case out <- Result{DownloadImage(ImageURLFromImageInfo(j), rLimit, signal), j.Name, no, j.HasWebp == 1}:
		case <-ctrl:
			return
		}
	}
}

var Gallery_ID = flag.String("Gallery_ID", "", "Hitomi.la Gallery ID")
var Gallery_Name = flag.String("Gallery_Name", "", "Hitomi.la Gallery name")
var Do_Compression = flag.Bool("Do_Compression", true, "Compress downloaded files if true")
var HTTPSvr = flag.Bool("HTTPSvr", false, "Start HTTP Server")
var RetryLimit = flag.Int("Retry_Limit", 3, "Limit of image download retry")
var Socks5 = flag.String("Socks5_Proxy", "", "Socks5 Proxy address")

var Client fasthttp.Client

func init() {
	flag.StringVar(Gallery_ID, "i", "", "Hitomi.la Gallery ID")
	flag.StringVar(Gallery_Name, "n", "", "Hitomi.la Gallery Name")
	flag.BoolVar(Do_Compression, "c", true, "Compress downloaded files if true")
	flag.BoolVar(HTTPSvr, "s", false, "Start HTTP Server")
	flag.IntVar(RetryLimit, "r", 3, "Limit of image download retry")
	flag.StringVar(Socks5, "socks", "", "Socks5 Proxy address")
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic:", r)
		}
	}()

	flag.Parse()
	if *Gallery_ID == "" {
		fmt.Println("<Commands>")
		fmt.Println("-i : Gallery ID")
		fmt.Println("-n : Gallery Name")
		fmt.Println("-c : Compression")
		fmt.Println("-s : Start HTTP Server")
		fmt.Println("-r : Limit of image download retry")
		fmt.Println("-socks : Socks5 proxy address")
		os.Exit(1)
	}
	if *Gallery_Name == "" {
		*Gallery_Name = *Gallery_ID
	}

	if *Socks5 != "" {
		Client.Dial = fasthttpproxy.FasthttpSocksDialer(*Socks5)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("using", runtime.GOMAXPROCS(0), "CPU(s)")

	fmt.Println("Gallery ID :", *Gallery_ID)
	fmt.Println("Gallery Name :", *Gallery_Name)
	fmt.Println("Compression :", *Do_Compression)
	fmt.Println("Start HTTP Server :", *HTTPSvr)
	fmt.Println("Download retry limit :", *RetryLimit)
	fmt.Println("Socks5 proxy address :", *Socks5)

	fmt.Println("fetching image list")
	img_lst := GetImageNamesFromID(*Gallery_ID)
	num_lst := len(img_lst)
	fmt.Println("fetched", num_lst, "images")

	var archiveFile *os.File
	var zipWriter *zip.Writer

	if *Do_Compression {
		//init zip archiver
		archiveFile, err := os.OpenFile(
			*Gallery_Name+".zip",
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			os.FileMode(0644))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		zipWriter = zip.NewWriter(archiveFile)
	} else {
		os.Mkdir(*Gallery_Name, 0777)
	}

	ctrl := make(chan struct{})
	jobs := make(chan ImageInfo)
	out := make(chan Result)
	signals := make(chan string)

	var wg sync.WaitGroup
	NumWorkers := 10
	wg.Add(NumWorkers)

	go func() {
		for {
			fmt.Println(<-signals)
		}
	}()

	for i := 0; i < NumWorkers; i++ {
		go func(n int) {
			DownloadWorker(n, *RetryLimit, signals, ctrl, jobs, out)
			wg.Done()
		}(i)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	go func() {
		for _, work := range img_lst {
			jobs <- work
		}
		close(jobs)
	}()

	count := 0
	for r := range out {
		count++

		if r.IsWebp {
			img, ext, err := image.Decode(bytes.NewBuffer(r.Image))
			if err != nil {
				log.Println(err)
			}

			if ext != "webp" {
				log.Printf("Image extension mismatch: %s != webp", ext)
			}

			var iBuffer bytes.Buffer
			err = jpeg.Encode(&iBuffer, img, &jpeg.Options{Quality: 100})
			if err != nil {
				log.Println("Encode Error:", err)
			}

			r.Image = iBuffer.Bytes()
		}

		if *Do_Compression {
			var f io.Writer
			var err error
			if r.IsWebp {
				f, err = zipWriter.Create(r.ImgName + ".jpg")
			} else {
				f, err = zipWriter.Create(r.ImgName)
			}
			if err != nil {
				fmt.Println(err)
			}
			_, err = f.Write(r.Image)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			var err error
			if r.IsWebp {
				err = ioutil.WriteFile(*Gallery_Name+"/"+r.ImgName+".jpg", r.Image, os.FileMode(0644))
			} else {
				err = ioutil.WriteFile(*Gallery_Name+"/"+r.ImgName, r.Image, os.FileMode(0644))
			}
			if err != nil {
				fmt.Println(err)
			}
		}
		fmt.Printf("[worker %d] downloaded %s\n", r.WK_ID, r.ImgName)

		if count == num_lst {
			close(ctrl)
		}
	}

	if *Do_Compression {
		zipWriter.Close()
		archiveFile.Close()
	}

	if *HTTPSvr == true {
		fmt.Println("HTTP Server started. Press Ctrl+C to exit")
		LnsCurrentDirectory()
	}
}
