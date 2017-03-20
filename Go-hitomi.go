package main

import(
	"bufio"
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"encoding/json"
	"bytes"
	"runtime"
	"sync"
)

type ImageInfo struct{
	Width uint `json:"width"`
	Name string `json:"name"`
	Height uint `json:"height"`
}

func GetImageNames(GalleryID string) []string{
	fmt.Println("starting downloading")
	resp,err:=http.Get("https://hitomi.la/galleries/"+GalleryID+".js")
	if err!=nil{
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body,err:=ioutil.ReadAll(resp.Body)
	if err!=nil{
		fmt.Println(err)
	}
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

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("using",runtime.GOMAXPROCS(0),"CPU(s)")

	scanner:=bufio.NewScanner(os.Stdin)
	fmt.Print("Enter gallery ID: ")
	scanner.Scan()
	galleryid:=scanner.Text()
	os.Mkdir(galleryid,0777)
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
				data,err:=http.Get("https://a.hitomi.la/galleries/"+galleryid+"/"+Imagename)
				if err!=nil{
					fmt.Println(err)
				}
				defer data.Body.Close()
				img,err:=ioutil.ReadAll(data.Body)
				err=ioutil.WriteFile(galleryid+"/"+Imagename,img,os.FileMode(644))
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
}