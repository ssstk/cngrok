package server

import (
	"flag"
)

type Options struct {
	httpAddr   string
	httpsAddr  string
	tunnelAddr string
	domain     string
	tlsCrt     string
	tlsKey     string
	logto      string
	loglevel   string
}

func parseArgs() *Options {
	httpAddr := flag.String("httpAddr", ":80", "HTTP连接端口，禁用空字符串")
	httpsAddr := flag.String("httpsAddr", ":443", "HTTPS连接端口，禁用空字符串")
	tunnelAddr := flag.String("tunnelAddr", ":4443", "ngrok客户端连接端口，禁用空字符串")
	domain := flag.String("domain", "yilu.ml", "承载隧道的域名")
	tlsCrt := flag.String("tlsCrt", "", "TLS证书文件的路径")
	tlsKey := flag.String("tlsKey", "", "TLS密钥文件的路径")
	logto := flag.String("log", "stdout", "将日志消息写入此文件。 'stdout'和'none'有特殊意义")
	loglevel := flag.String("log-level", "INFO", "要记录的消息级别。 其中之一: DEBUG, INFO, WARNING, ERROR")
	flag.Parse()

	return &Options{
		httpAddr:   *httpAddr,
		httpsAddr:  *httpsAddr,
		tunnelAddr: *tunnelAddr,
		domain:     *domain,
		tlsCrt:     *tlsCrt,
		tlsKey:     *tlsKey,
		logto:      *logto,
		loglevel:   *loglevel,
	}
}
