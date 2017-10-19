package downloader

import(
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
		case out <- Result{DownloadImage("https://a.hitomi.la/galleries/"+GalleryId+"/"+j),j,no}:
		case <-ctrl:
			return
		}
	}
}