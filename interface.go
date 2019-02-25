package main

import "context"

type AutoFtpInterface interface {
	connect(info *FtpInfo, ctx context.Context) error
	handle(pathInfo *PathObject)
	close() error
}
