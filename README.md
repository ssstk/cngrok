# 原文地址来自我的博客，欢迎访问：[https://www.hexcode.cn/article/show/raspberry-ngrok](https://www.hexcode.cn/article/show/raspberry-ngrok)

>场景需求：家里的几台树莓派通过家用WIFI路由器上网，虽然装了Teamviewer可以远程穿透内网控制图形界面，但远程时屏幕分辨率太小，体验不佳，于是想让树莓派上的SSH也能拥有穿透内网的功能。

## 前言，关于Ngrok

查阅了很多资料，发现大多数内网穿透功能的实现都与一款Go语言编写的Ngrok项目相关，Ngrok是一整套的服务器，客户端解决方案。因为其是由Go语言编写的，所以天生具备很强的跨平台能力（但目前基本上所有资料都指向在Linux上部署该服务器端）。博主使用的是阿里云的Windows VPS，并且这段时间正好也在使用Go编程，因此这篇文章将具体介绍如何在Windows上部署Ngrok项目，并且在家用树莓派上部署客户端，实现树莓派的SSH内网穿透。

关于Ngrok，值得注意的有以下几点：
- Ngrok并不仅仅用来22端口的SSH内网穿透，通过简单配置，它可以将遵循TCP/UDP协议的多个端口进行内网穿透。
- Ngrok是由一台服务器端和一个或者多个客户端组成的体系。
- Ngrok需要一台部署在公网固定IP上的服务器，最好有可正常指向的域名，Ngrok的服务器端就部署在上面，如本次实验的阿里云主机。
- Ngrok的客户端装在需要映射的各个没有公网IP的机器上，比如本次实验的树莓派。
- Ngrok服务器会在接受到Ngrok客户端请求时分配固定或者随机的端口号，将客户端请求的端口与之映射，从而达到内网穿透的目的。
- 用户在远程使用SSH客户端进行连接时，不需要任何Ngrok程序，对终端用户来说，Ngrok是透明的。
- [Ngrok](https://ngrok.com/)自身有一台欧洲的公网服务器，供没有VPS的用户注册使用，该服务器并不是以分配端口的方式映射，而是以分配子域名的方式，随机子域名是免费使用的，固定子域名映射是收费的。
- 普通玩家可以直接使用[Ngrok的官网](https://ngrok.com/)服务器，缺点就是有一点点慢，而且每次客户端重启后子域名会随之变化，无法固定使用。
- 国内也有一些商家提供了Ngrok服务，比如[Sunny-Ngrok](https://www.ngrok.cc/)，同样是收费的，速度可能快一点。

Ngrok源代码是开源的，部署在[Git](https://github.com/mamboer/ngrok)上，但是博主使用下来略觉繁琐，涉及到很多自认为没必要使用的技术，后期我重新在Git上发布了我重构后的[Ngrok项目](https://ngrok)，配有[码云同步镜像](https://gitee.com/newflydd/ngrok)，以供其他朋友直接使用诸如`go build`或者`go install`简单命令进行跨平台编译和安装。

为了致敬原作者，我们对这些高端的Golang编程技术做一个简单的了解，对Go语言没有兴致的朋友可以直接跳过下面这段:
- _go-bindata_ 技术，该技术是一套将任意资源文件转成二进制数据反向生成到Go代码供hash调用的技术，Ngrok的源码利用这一技术将公网证书和密钥，以及站点的HTML生成到go代码中去了，使其变成了一堆byte数组，在程序中hash调用。这么做固然有它的好处，但缺点也显而易见，因为这套技术要提前用一个项目的程序生成另一个项目的Go代码，所以编译起来难免繁琐，于是Ngrok提供了makefile文件以供用户make编译，这就有点尴尬了，本来好好的跨平台Go语言项目，被make和makefile活生生憋成了只能Linux使用的了。博主初期使用时硬着头皮安装了`Cygwin`，以及其中的`make`命令，个中的苦逼无以言表，索性后来了解原理后，将官方的Ngrok重构了一遍，避免了使用几乎是专属于Linux的make指令。
- _build -tags_ 技术，通过Ngrok项目，我第一次知道Go的这个特性，可以在代码顶端加上`// +build !release`等标记，注明编译时的模式，比如前面这种表达方式，就是在非*release*编译模式下才会编译这个文件，然后使用`go build -tags 'release'`或者`go build -tags 'debug'`来控制是否编译这个代码文件。这个特性确实灵活，可以在项目中编写不同的编译模式下的代码，缺点呢，就是没有详细文档的情况下，第三方用户并不知道有哪些编译模式，加大了源码本地编译的复杂度，所以这也是原版Ngrok项目使用makefile文件的原因之一。
- _[equinox.io](https://www.equinox.io)_ 技术，这项技术可以让Go代码在编译和运行期间保持更新，代码作者更新代码后，能够在第三方用户实时更新，具体没有了解，我看原版Ngrok在进行*release*编译时增加了很多*equinox.io*令牌密钥，以连接远程仓库，保证代码更新。

总之，上面的这些特性，虽然优点多多，但使用起来难免加大了难度，并且牺牲了Go优良的跨平台特性，这是我不能忍的，于是我重新构造了Ngrok项目，旨在让更多玩家，在众多平台上简单轻松的编译Ngrok项目。

## 言归正传，准备工作
下面我们言归正传，从零开始部署Ngrok，说说准备工作：

### 硬件部分
- 一台公网IP服务器，本实验采用Windows系统，拥有域名hexcode.cn。
- 一台装有RASPBIAN操作系统的树莓派，用来部署Ngrok客户端，使其可以SSH内网穿透
- 一台Windows PC，开发环境

### 软件部分
- [Win32 OpenSSL v1.1.0f Light](http://slproweb.com/download/Win32OpenSSL_Light-1_1_0f.exe)证书生成程序
- [MobaXterm](https://download.mobatek.net/10420170816103227/MobaXterm_Portable_v10.4.zip)几乎是最好的SSH客户端
- [go1.9.2.windows-amd64.msi](https://redirector.gvt1.com/edgedl/go/go1.9.2.windows-amd64.msi)Go语言编译器，我现在用得最多的语言，一股子的亲切感，交叉编译Linux，ARM，X86等平台的目标程序就跟玩一样
- [Ngrok源码](https://ngrok)这是我自己重构的Ngrok项目，原版的要想用起来还得安装Cygwin等一堆工具，现在非常纯净，拿到手直接用最简单的命令编译，各平台通吃。这份代码不需要手工download，下面会介绍直接用命令获取。
- [Git-windows-amd64](https://github.com/git-for-windows/git/releases/download/v2.14.3.windows.1/Git-2.14.3-64-bit.exe)这个就不赘述了，程序猿都知道。

## 从零开始操作
### 证书生成
Ngrok需要使用SSL证书确保穿透过程中的通信是加密安全的，因此需要SSL证书，值得庆幸的是并不强制去购买CA认证机构颁发的证书，普通用户直接使用OpenSSL工具就可以自行生成证书。方法如下：
- 下载上述准备工作软件部分的SSL证书生成程序，安装后，将安装路径下的bin目录添加到环境变量，方便cmd直接运行`openssl.exe`。
- 任意路径打开cmd，执行下面命令：(注意，将两处MY_DOMAIN全部替换成自己的域名，比如我的hexcode.cn)
```
openssl genrsa -out rootCA.key 2048
openssl req -x509 -new -nodes -key rootCA.key -subj "/CN=MY_DOMAIN" -days 5000 -out rootCA.pem
openssl genrsa -out device.key 2048
openssl req -new -key device.key -subj "/CN=MY_DOMAIN" -out device.csr
openssl x509 -req -in device.csr -CA rootCA.pem -CAkey rootCA.key -CAcreateserial -out device.crt -days 5000

cp rootCA.pem ngrokroot.crt
cp device.crt snakeoil.crt
cp device.key snakeoil.key
```
- 全部执行完毕后，会在当前路径下生成6个文件，我们服务器端会用到其中的 _snakeoil.crt_ , _snakeoil.key_ ,客户端会用到 _ngrokroot.crt_ 文件，先不管它们，留到下面使用。

### 编译Windows平台下的服务器端程序
得益于Go的众多优秀的特性，让一份跨平台的代码能够快速地适配不同平台编译和安装。相比其他语言不同的版本，不同的编译方式，编译工具，甚至有些是超级重量级的，比如JAVA，各个平台下的JDK少说也要300M，Go的编译器全平台覆盖，并且只有90M左右，非常轻巧，并且只需要在某一平台下下载一次，就可以轻松靠着两个环境变量编译出全平台覆盖的可执行程序，交叉编译从未如此简单过。

一聊起Go的优点，我就差点收不住。我们来实际体验一下吧，在上方准备工作软件部分下载Go的Windows平台编译器，按照提示安装，设置Windows环境变量`PATH`路径添加Go安装目录下的bin文件夹，方便cmd中直接使用`go`命令，为了还能直接使用`go get`命令，我们需要安装Git，下载地址在上方，下载回来直接根据提示全部以默认选项安装即可。
另外需要配置`GOPATH`,`GOROOT`两个环境变量，这两个并不是必须的，但为了减少麻烦还是建议设置，其中`GOROOT`变量指向Go的安装路径，比如`C:\GoRoot`，`GOPATH`是我们的工作区路径，随意填写，写完了就在此路径下编写程序就行了，当然如果你仅仅用来完成这个项目，以后不会学习Go也不会编写Go语言的代码，这个路径对你来说仅仅是接下来自动下载源代码的保存路径而已。这里我们简单设置成`C:\GoPath`。
如图所示：
[![](http://ocsy1jhmk.bkt.clouddn.com/42d34d36-ea54-432c-a0f0-ae53befa1adb.png)](http://)

OK，接下来正式下载，编译，和安装Ngrok的服务器端。
你一定会以为这一系列操作很复杂，其实对于Go来说仅仅是一条命令行的指令：
```
go get ngrok/main/ngrokd
```
就这么简单，就这么神奇，简单到你不管在什么路径下执行这行命令都没有问题。这行命令的背后可不简单，go首先会去下载我上传到git上的Ngrok代码，然后试着编译，编译过程中它会发现Ngrok项目还七零八落地依赖了其他各个项目，于是Go用递归的方式再去执行go get命令，直到将所有依赖通通下载回来，最终它成功编译后，将生成的目标程序存放在`%GOPATH%\bin`目录下，默认情况下它是根据当前机器的环境来生成可执行文件，如果做一些简单的调整，它即可以交叉编译不同平台下的可执行文件。

执行过程可能会有些漫长，跟网络环境相关，建议使用全局科学上网工具。
成功编译后，在`%GOPATH%\bin`目录下会出现`ngrokd.exe`可执行程序，这就是你要上传到Windows服务器的程序，大小在8M左右，同时你需要上传第一步生成SSL证书中的 _snakeoil.crt_ , _snakeoil.key_ 两份文件到同目录。`ngrokd.exe`双击直接运行是没有意义的，它需要配有参数，因此我们编写bat脚本来方便运行（将下面的MY_DOMAIN替换成自己的域名，比如我的hexcode.cn）：
```
ngrokd.exe -domain="MY_DOMAIN" -log ./ngrokd.log &
```
将上面脚本保存为`run.bat`，双击运行是一个命令行界面，同目录下会出现ngrokd.log，打开看一眼，如果出现以下字样，就说明服务器端程序已经正常运行了：
```
Listening for public http connections on [::]:80
Listening for public https connections on [::]:443
Listening for control and proxy connections on [::]:4443
```
运行时的画面如下：
[![](http://ocsy1jhmk.bkt.clouddn.com/4f9ff28d-f0eb-4046-9b5b-1c1ea15a183d.png)](http://)

### 编译Linux ARM平台下的树莓派客户端
试想一下，如果没有交叉编译，我们该如何在树莓派上安装Ngrok客户端：我们必须同样在树莓派上安装Go的Linux ARM版的编译器；RASPBIAN系统已经安装了git，所以我们不需要独立安装git；我们还得用go get指令下载和编译Ngrok的源码；openssl指令不用装了，因为证书已经生成了。

虽然也不是特别复杂，但还是挺费时间的，而且还会占用树莓派的资源，平白无故地要去安装接近90M的Go编译环境。得益于Go的强大，交叉编译发挥了巨大作用，我们在Windows X86 AMD64上就能编译出能在Linux ARM上跑的可执行文件。

打开一个新的cmd命令行，任意路径，执行下面命令：
```
set GOOS=linux
set GOARCH=arm
go get ngrok/main/ngrok
```
OK，就这么简单，就这么神奇，Windows AMD64上的Go利用简单的两个环境变量，直接构造出了Linux ARM上的可执行文件，多么伟大的语言。。。
GOOS GOARCH的组合可以参照：

| GOOS | GOARCH |
| ------------ | ------------ |
| darwin | 386 |
| darwin | amd64 |
| freebsd | 386 |
| freebsd | amd64 |
| linux | 386 |
| linux | amd64 |
| linux | arm |
| windows | 386 |
| windows | amd64 |

### 在树莓派中运行Ngrok客户端程序
Go将生成的Linux平台下的程序保存在`%GOPATH%\bin\linux_arm`目录下，名为`ngrok`，我们需要将其copy到树莓派中。
我们使用超级强大的终端工具MobaXterm来连接树莓派并上传ngrok，MobaXterm的下载地址在本文上方，下载后按照提示就能安装，因为目前树莓派的SSH还不具备内网穿透功能，因此下面操作必须在局域网中完成。

使用MobaXterm建立与树莓派的SSH连接，这里并不是重点，相关操作如果不清楚可以自行百谷。
MobaXterm具有强大的文件上传功能，直接将使用下面图例中的按钮将Windows本地的`ngrok`客户端程序上传到指定目录，同时将我们第一步SSL证书环节生成的 _ngrokroot.crt_ 文件上传进去。
[![](http://ocsy1jhmk.bkt.clouddn.com/45f62389-697d-4056-ae86-1049b26b05e8.png)](http://)

我这里操作的目录为：`/home/pi/workspace/go-apps/ngrok/`，此时上传上去的`ngrok`文件不具备执行能力，需要使用下面命令为其添加执行权限：
```
chmod a+x ngrok
```
同样，ngrok程序不可以直接运行，也需要参数，并且还需要一个配置文件，我们执行下面操作新建配置文件，并编写sh脚本，以便直接运行。
```
nano ngrok.cfg

```
新建ngrok.cfg配置文件，并用nano文本编辑器打开，输入以下文本（将MY_DOMAIN替换成自己的域名，比如我的hexcode.cn）：
```
server_addr: "MY_DOMAIN:4443"
trust_host_root_certs: false

tunnels:
    ssh:
        remote_port: 50000
        proto:
            tcp: 22
```
我这里仅仅需要让SSH的22端口具备穿透能力，因此我只配置了ssh任务，指定了远程端口固定为50000，映射本地的SSH22端口。Ctrl+O并回车保存，Ctrl+X退出文本编辑器。
```
nano run.sh
```
新建run.sh脚本文件，并用文本编辑器打开，输入以下文本：
```
nohup ./ngrok -log ./ngrok.ssh.log -config ./ngrok.cfg start ssh &
```
Ctrl+O并回车保存，Ctrl+X退出文本编辑器。
```
chmod a+x run.sh
```
为run.sh添加可执行权限。OK，大功告成，运行`./run.sh`即可建立树莓派与公网服务器的通信隧道，将其22端口映射到了公网服务器的50000端口，我们接下来就可以在世界任何角落，使用任何SSH工具，连接自己公网服务器的50000端口，访问家中的树莓派了，下图用MobaXterm做一个测试：
[![](http://ocsy1jhmk.bkt.clouddn.com/c1593868-7b87-4abe-b5da-e9ddc915806c.png)](http://)

[![](http://ocsy1jhmk.bkt.clouddn.com/c89d0dbb-6e13-4a14-94f2-d7aa1becb232.png)](http://)


