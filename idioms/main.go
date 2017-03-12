package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/golang/glog"
)

var g_flags struct {
	path string

	listen string

	gcnum int
}

func init() {
	flag.StringVar(&g_flags.path, "input",
		"/the/path/does/not/exists.txt",
		"路径, 其指向着的文件将会作为 LoadIdiomGraph() 的参数")

	flag.StringVar(&g_flags.listen, "listen", ":12138",
		"$IP:$PORT, 其值将作为 http.Server 的 listen 参数")

	flag.IntVar(&g_flags.gcnum, "gcnum", 3000, "int; 数字越大, gc 频率越低")
}

var g_idiom_graph *IdiomGraph
var g_http_server *http.Server

func sigThreadMain() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)

	<-sigchan

	glog.Info("catch sigint")
	err := g_http_server.Shutdown(context.Background())
	if err != nil {
		glog.Error("http server shutdown; err: ", err)
	}
	return
}

func main() {
	flag.Parse()

	defer glog.Flush()

	ifile, err := os.Open(g_flags.path)
	if err != nil {
		panic(err)
	}

	glog.Info("准备 LoadIdiomGraph")
	g_idiom_graph, err = LoadIdiomGraph(ifile, g_flags.gcnum)
	if err != nil {
		panic(err)
	}
	glog.Info("http server run")

	/* 这里存在两个问题.
	1. 测试发现当调用 http.Server.Shutdown() 之后仍然可以正常运行 http.Server.ListenAndServe().
	   因此若在 ListenAndServe() 之前收到了 SIGINT 信号, 可能会导致 Shutdown() 先于 ListenAndServe()
	   之前执行.

	2. 当 Shutdown() 返回时, ListenAndServe() 所处 goroutine 可能还没有结束, 也就是说 main goroutine
	   可能先退出, 不知道这时的设定是什么.
	*/
	g_http_server = &http.Server{Addr: g_flags.listen}
	go g_http_server.ListenAndServe()
	sigThreadMain()

	return
}
