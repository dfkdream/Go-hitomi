package main

import(
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"encoding/json"
	"bytes"
	"runtime"
	"sync"
	"flag"
	"archive/zip"
)

type ImageInfo struct{
	Width uint `json:"width"`
	Name string `json:"name"`
	Height uint `json:"height"`
}

func GetImageNames(GalleryID string) []string{
	fmt.Println("starting downloading")
	resp,_:=http.Get("https://hitomi.la/galleries/"+GalleryID+".js")
	defer resp.Body.Close()
	body,_:=ioutil.ReadAll(resp.Body)
	body=bytes.Replace(body,[]byte("var galleryinfo = "),[]byte(""),-1)
	fmt.Println("replaced")
	var ImageInfo []ImageInfo
	fmt.Println("starting parsing")
	json.Unmarshal(body,&ImageInfo)
	fmt.Println("parsing finished")
	var ImageNames []string
	for _,Info := range ImageInfo{
		ImageNames=append(ImageNames,Info.Name)
	}

	return ImageNames
}

func httpsvr(){
	http.Handle("/",http.StripPrefix("/",http.FileServer(http.Dir("."))))

	http.ListenAndServe(":80",nil)
}

var Gallery_ID=flag.String("Gallery_ID","","Hitomi.la Gallery ID")
var Gallery_Name=flag.String("Gallery_Name","","Hitomi.la Gallery name")
var Do_Compression=flag.Bool("Do_Compression",true,"Compress downloaded files if true")
var HTTPSvr=flag.Bool("HTTPSvr",false,"Start HTTP Server")
var mutex=new(sync.Mutex)

func DownloadAndSave(id string, name string, comp bool){
	var archiveFile *os.File
	var zipWriter *zip.Writer
	if comp{
		//init zip archiver
		archiveFile,err:=os.OpenFile(
			name+".zip",
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			os.FileMode(0644))
		if err!=nil{
			fmt.Println(err)
			os.Exit(1)
		}
		zipWriter=zip.NewWriter(archiveFile)
	}else{
		os.Mkdir(name,0777)
	}
	ImageNames:=GetImageNames(id)
	fmt.Println(ImageNames)

	wg:=new(sync.WaitGroup)

	buff:=make(chan string,5)

	for i:=0;i<5;i++{
		ImageNames=append(ImageNames,"end")
		wg.Add(1)
		go func(workerID int){
			for{
				Imagename:=<-buff
				if Imagename=="end"{
					wg.Done()
					return
				}
				data,_:=http.Get("https://a.hitomi.la/galleries/"+id+"/"+Imagename)
				defer data.Body.Close()
				img,err:=ioutil.ReadAll(data.Body)
				if comp{
					mutex.Lock()
					f,err:=zipWriter.Create(Imagename)
					if err!=nil{
						fmt.Println(err)
					}
					_,err=f.Write(img)
					if err==nil{
						fmt.Println("[worker",workerID,"] downloaded",Imagename)
					}else{
						fmt.Println(err)
					}
					mutex.Unlock()
				}else{
					err=ioutil.WriteFile(name+"/"+Imagename,img,os.FileMode(0644))
					if err==nil{
						fmt.Println("[worker",workerID,"] downloaded",Imagename)
					}else{
						fmt.Println(err)
					}
				}		
			}
		}(i)
	}
	for _,imagename := range ImageNames{
		buff<-imagename
	}
	wg.Wait()

	if *Do_Compression{
		zipWriter.Close()
		archiveFile.Close()
	}
}

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

	DownloadAndSave(*Gallery_ID,*Gallery_Name,*Do_Compression)

	if *HTTPSvr==true{
		fmt.Println("HTTP Server started. Press Ctrl+C to exit")
		httpsvr()
	}
}