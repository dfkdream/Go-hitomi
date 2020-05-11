# Go-hitomi
### Simple *hitomi.la* downloader written in Golang

#### Build
1. `git clone https://github.com/dfkdream/Go-hitomi.git`
2. `go build`

Or you can download latest Windows x64 build [here](https://github.com/dfkdream/Go-hitomi/releases/latest)

#### Commands
* `-i Gallery_ID(int)`: Set gallery ID **(required)**
* `-n Gallery_name(str)`: Set gallery name(filename) **(optional)**
* `-c Compression(bool)`: Compress files if true **(optional, default:true)**
* `-s HTTPSvr(bool)`: Start HTTP file server in current directory port 80 **(optional, default:false)** 
* `-r RetryLimit(int)`: Limit of image download retry **(optional, default:3)**
* `-socks Socks5_Proxy(string)`: Socks5 proxy address **(optional)**

#### Basic how to use
1. Run command `Go-hitomi -i [Gallery_ID] -n [Gallery_name]`
2. Image will be downloaded at `\Gallery_name.zip`.
