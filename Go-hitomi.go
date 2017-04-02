package main

import(
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"os/exec"
	"encoding/json"
	"bytes"
	"runtime"
	"sync"
	"flag"
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
var Remove_Origin=flag.Bool("Remove_Origin",true,"Remove original files if true")
var HTTPSvr=flag.Bool("HTTPSvr",false,"Start HTTP Server")

func init(){
	flag.StringVar(Gallery_ID,"i","","Hitomi.la Gallery ID")
	flag.StringVar(Gallery_Name,"n","","Hitomi.la Gallery Name")
	flag.BoolVar(Do_Compression,"c",true,"Compress downloaded files if true")
	flag.BoolVar(Remove_Origin,"r",true,"Remove original files if true")
	flag.BoolVar(HTTPSvr,"s",false,"Start HTTP Server")
}
func main() {
	flag.Parse()
	if (*Gallery_ID==""){
		fmt.Println("<Commands>")
		fmt.Println("-i : Gallery ID")
		fmt.Println("-n : Gallery Name")
		fmt.Println("-c : Compression")
		fmt.Println("-r : Remove original files")
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
	fmt.Println("Remove original files :",*Remove_Origin)
	fmt.Println("Start HTTP Server :",*HTTPSvr)

	galleryid:=*Gallery_ID
	os.Mkdir(*Gallery_Name,0777)
	ImageNames:=GetImageNames(galleryid)
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
				data,_:=http.Get("https://a.hitomi.la/galleries/"+galleryid+"/"+Imagename)
				defer data.Body.Close()
				img,err:=ioutil.ReadAll(data.Body)
				err=ioutil.WriteFile(*Gallery_Name+"/"+Imagename,img,os.FileMode(644))
				if err==nil{
					fmt.Println("[worker",workerID,"] downloaded",Imagename)
				}else{
					fmt.Println(err)
				}
		
			}
		}(i)
	}
	for _,imagename := range ImageNames{
		buff<-imagename
	}
	wg.Wait()

	fmt.Println("Compressing...")

	if *Do_Compression==true{
		err:=exec.Command("7z","a",*Gallery_Name+".zip","./"+*Gallery_Name+"/*").Run()
		if err!=nil{
			fmt.Println(err)
		}
	}

	if *Remove_Origin==true{
		err:=os.RemoveAll(*Gallery_Name)
		if err!=nil{
			fmt.Println(err)
		}
	}

	if *HTTPSvr==true{
		fmt.Println("HTTP Server started. Press Ctrl+C to exit")
		httpsvr()
	}
}