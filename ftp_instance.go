package main

import (
	"context"
	log "github.com/wuzuoliang/micro-backend/MyLogger"
	"time"
)

func initInstance(ins *Instance) {
	if ins.FtpInfo.TransType != TYPE_SFTP && ins.FtpInfo.TransType != TYPE_FTP {
		log.Fatal("[initInstance]", "NO SUCH TYPE", ins.FtpInfo.TransType)
	}
	var client = new(SFtpClient)
	for i := 0; i < len(ins.FtpPaths); i++ {
		if ins.FtpInfo.TransType == TYPE_SFTP {
		RetrySFTP:
			err := client.connect(&ins.FtpInfo, context.WithValue(context.Background(), "PATH_ID", i))
			if err != nil {
				log.Error("[SFTP connect] fail", err)
				time.Sleep(time.Minute)
				goto RetrySFTP
			}
			go client.handle(&ins.FtpPaths[i])
		} else if ins.FtpInfo.TransType == TYPE_FTP {
			// TODO make ftp client concurrently
			//for k := 0; k < ins.FtpPaths[i].MultiGo; k++ {
			clients := new(FtpClient)
		RetryFTP:
			err := clients.connect(&ins.FtpInfo, context.WithValue(context.Background(), "PATH_ID", i))
			//err := clients.connect(&ins.FtpInfo, context.WithValue(context.WithValue(context.Background(), "PATH_ID", i), "THREAD_ID", k))
			if err != nil {
				log.Error("[FTP connect] fail", err)
				time.Sleep(time.Minute)
				goto RetryFTP
			}
			go clients.handle(&ins.FtpPaths[i])
			//}
		}
	}
}
