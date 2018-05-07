package main

import(
	"fmt"
	"os"
	"runtime"
	"sync"
	"flag"
	"archive/zip"
	"io/ioutil"
	"net/http"
	"github.com/valyala/fasthttp"
	"encoding/json"
	"bytes"
)

type ImageInfo struct{
	Width uint `json:"width"`
	Name string `json:"name"`
	Height uint `json:"height"`
}

type Result struct{
	Image []byte
	ImgName string
	WK_ID int
}

func GetImageNamesFromID(GalleryID string) []string{
	_,resp,_:=fasthttp.Get(nil,"https://hitomi.la/galleries/"+GalleryID+".js")
	resp=bytes.Replace(resp,[]byte("var galleryinfo = "),[]byte(""),-1)
	var ImageInfo []ImageInfo
	json.Unmarshal(resp,&ImageInfo)
	var ImageNames []string
	for _,Info := range ImageInfo{
		ImageNames=append(ImageNames,Info.Name)
	}
	return ImageNames
}

func LnsCurrentDirectory(){
	http.Handle("/",http.StripPrefix("/",http.FileServer(http.Dir("."))))

	http.ListenAndServe(":80",nil)
}

func DownloadImage(url string)[]byte{
	if stat,img,err:=fasthttp.Get(nil,url);stat==200&&err==nil{
		return img
	}
	return nil
}

func DownloadWorker(no int, GalleryId string, ctrl <-chan struct{}, jobs <-chan string, out chan<- Result){
	for j:=range jobs{
		select{
		case out <- Result{DownloadImage("https://ba.hitomi.la/galleries/"+GalleryId+"/"+j),j,no}:
		case <-ctrl:
			return
		}
	}
}

var Gallery_ID=flag.String("Gallery_ID","","Hitomi.la Gallery ID")
var Gallery_Name=flag.String("Gallery_Name","","Hitomi.la Gallery name")
var Do_Compression=flag.Bool("Do_Compression",true,"Compress downloaded files if true")
var HTTPSvr=flag.Bool("HTTPSvr",false,"Start HTTP Server")

func init(){
	flag.StringVar(Gallery_ID,"i","","Hitomi.la Gallery ID")
	flag.StringVar(Gallery_Name,"n","","Hitomi.la Gallery Name")
	flag.BoolVar(Do_Compression,"c",true,"Compress downloaded files if true")
	flag.BoolVar(HTTPSvr,"s",false,"Start HTTP Server")
}

func main() {
	flag.Parse()
	if (*Gallery_ID==""){
		fmt.Println("<Commands>")
		fmt.Println("-i : Gallery ID")
		fmt.Println("-n : Gallery Name")
		fmt.Println("-c : Compression")
		fmt.Println("-s : Start HTTP Server")
		os.Exit(1)
	}
	if (*Gallery_Name==""){
		*Gallery_Name=*Gallery_ID
	}

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("using",runtime.GOMAXPROCS(0),"CPU(s)")

	fmt.Println("Gallery ID :",*Gallery_ID)
	fmt.Println("Gallery Name :",*Gallery_Name)
	fmt.Println("Compression :",*Do_Compression)
	fmt.Println("Start HTTP Server :",*HTTPSvr)

	fmt.Println("fetching image list")
	img_lst:=GetImageNamesFromID(*Gallery_ID)
	num_lst:=len(img_lst)
	fmt.Println("fetched",num_lst,"images")

	var archiveFile *os.File
	var zipWriter *zip.Writer

	if *Do_Compression{
		//init zip archiver
		archiveFile,err:=os.OpenFile(
			*Gallery_Name+".zip",
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			os.FileMode(0644))
		if err!=nil{
			fmt.Println(err)
			os.Exit(1)
		}
		zipWriter=zip.NewWriter(archiveFile)
	}else{
		os.Mkdir(*Gallery_Name,0777)
	}

	ctrl:=make(chan struct{})
	jobs:=make(chan string)
	out:=make(chan Result)

	var wg sync.WaitGroup
	NumWorkers:=10
	wg.Add(NumWorkers)

	for i:=0;i<NumWorkers;i++{
		go func(n int){
			DownloadWorker(n,*Gallery_ID,ctrl,jobs,out)
			wg.Done()
		}(i)
	}

	go func(){
		wg.Wait()
		close(out)
	}()

	go func(){
		for _,work:=range img_lst{
			jobs <- work
		}
		close(jobs)
	}()

	count:=0
	for r:=range out{
		count++

		if *Do_Compression{
			f,err:=zipWriter.Create(r.ImgName)
			if err!=nil{
				fmt.Println(err)
			}
			_,err=f.Write(r.Image)
			if err!=nil{
				fmt.Println(err)
			}
		}else{
			err:=ioutil.WriteFile(*Gallery_Name+"/"+r.ImgName,r.Image,os.FileMode(0644))
			if err!=nil{
				fmt.Println(err)
			}
		}
		fmt.Printf("[worker %d] downloaded %s\n",r.WK_ID,r.ImgName)

		if count==num_lst{
			close(ctrl)
		}
	}

	if *Do_Compression{
		zipWriter.Close()
		archiveFile.Close()
	}

	if *HTTPSvr==true{
		fmt.Println("HTTP Server started. Press Ctrl+C to exit")
		LnsCurrentDirectory()
	}
}
