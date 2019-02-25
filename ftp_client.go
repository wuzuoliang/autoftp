package main

import (
	"context"
	"github.com/shenshouer/ftp4go"
	log "github.com/wuzuoliang/micro-backend/MyLogger"
	"github.com/wuzuoliang/micro-backend/Tools/TypeCastTool"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type FtpClient struct {
	AutoFtpInterface
	client *ftp4go.FTP
	ctxs   context.Context
}

func (f *FtpClient) connect(info *FtpInfo, ctx context.Context) error {
	f.client = ftp4go.NewFTP(0) // 1 for debugging
	//connect
	_, err := f.client.Connect(info.Ip, info.Port, "")
	if err != nil {
		log.Error("The connection failed")
		return err
	}

	_, err = f.client.Login(info.User, info.Passwd, "")
	if err != nil {
		log.Error("The login failed")
		return err
	}
	f.ctxs = ctx
	log.Info("[FTP connect success]", "PATH_ID", ctx.Value("PATH_ID"))
	return nil
}

// this ftp lib not implement concurrent,so this handle is a sequence
func (f *FtpClient) handle(pathInfo *PathObject) {
	defer f.close()
	r, _ := regexp.Compile(pathInfo.CompareStr)
	if pathInfo.TransMode == TRANS_TYPE_DOWNLOAD {
		f.scanFileFromRemotePath(pathInfo, r)
	} else if pathInfo.TransMode == TRANS_TYPE_UPLOAD {
		f.scanFileFromLocalPath(pathInfo, r)
	} else {
		log.Error("[handle]", "no such trans_mode", pathInfo.TransMode)
	}
}
func (f *FtpClient) close() error {
	if f.client != nil {
		if _, err := f.client.Quit(); err != nil {
			log.Error("[FTP close error]", err)
			return err
		}
	} else {
		log.Info("[FTP no need close]")
	}
	return nil
}

// just not used now
func (f *FtpClient) consumer(pathInfo *PathObject) {
	for {
		select {
		case fileInfo := <-pathInfo.fileTransC:
			if pathInfo.TransMode == TRANS_TYPE_DOWNLOAD {
				f.download(pathInfo, fileInfo)
			} else if pathInfo.TransMode == TRANS_TYPE_UPLOAD {
				f.upload(pathInfo, fileInfo)
			}
		}
	}
}

func (f *FtpClient) scanFileFromRemotePath(pathInfo *PathObject, r *regexp.Regexp) {
	for {
		func() {
			var files []string
			files, err := f.client.Dir(pathInfo.SrcPath)
			if err != nil {
				log.Error("[scanFileFromRemotePath]", "Dir error", err)
				return
			}
			fileInfos := FilterFile(files)
			for _, v := range fileInfos {
				fileName := strings.Replace(v.Name, pathInfo.SrcPath, "", -1)
				if strings.HasPrefix(fileName, ".") {
					continue
				}
				fileInfo := FtpFileInfo{v.Name, int64(v.Size), v.Time}
				switch pathInfo.CompareMode {
				case COMPARE_ALL:
					f.download(pathInfo, fileInfo)
				case COMPARE_PRE:
					if strings.HasPrefix(fileName, pathInfo.CompareStr) {
						f.download(pathInfo, fileInfo)
					}
				case COMPARE_SUFF:
					if strings.HasSuffix(fileName, pathInfo.CompareStr) {
						f.download(pathInfo, fileInfo)
					}
				case COMPARE_REG:
					if r.MatchString(fileName) {
						f.download(pathInfo, fileInfo)
					}
				}
				time.Sleep(time.Duration(pathInfo.SleepTime) * time.Second)
			}
		}()
		time.Sleep(time.Minute)
	}
}

func (f *FtpClient) scanFileFromLocalPath(pathInfo *PathObject, r *regexp.Regexp) {
	for {
		err := filepath.Walk(pathInfo.SrcPath, func(fs string, info os.FileInfo, err error) error {
			if info == nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			fileName := strings.Replace(info.Name(), pathInfo.SrcPath, "", -1)
			if strings.HasPrefix(fileName, ".") {
				return nil
			}
			fileInfo := FtpFileInfo{info.Name(), info.Size(), info.ModTime()}
			switch pathInfo.CompareMode {
			case COMPARE_ALL:
				f.upload(pathInfo, fileInfo)
			case COMPARE_PRE:
				if strings.HasPrefix(fileName, pathInfo.CompareStr) {
					f.upload(pathInfo, fileInfo)
				}
			case COMPARE_SUFF:
				if strings.HasSuffix(fileName, pathInfo.CompareStr) {
					f.upload(pathInfo, fileInfo)
				}
			case COMPARE_REG:
				if r.MatchString(fileName) {
					f.upload(pathInfo, fileInfo)
				}
			}
			time.Sleep(time.Duration(pathInfo.SleepTime) * time.Second)
			return nil
		})
		if err != nil {
			log.Error("[scanFileFromLocalPath]", "upload error", err)
		}
		time.Sleep(time.Minute)
	}
}

func (f *FtpClient) download(pathInfo *PathObject, fileInfo FtpFileInfo) {
	log.Debug("[download]", "file", fileInfo.Name)
	_, err := f.client.Rename(pathInfo.SrcPath+fileInfo.Name, pathInfo.SrcPath+".lock_dw_"+fileInfo.Name)
	if err != nil {
		log.Error("[download]", "file lock error", err)
		return
	}
	err = f.client.DownloadFile(pathInfo.SrcPath+".lock_dw_"+fileInfo.Name, pathInfo.DestPath+".tmp_dw_"+fileInfo.Name, false)
	if err != nil {
		log.Error("[download]", "file", fileInfo.Name, "error", err)
		return
	}
	err = os.Rename(pathInfo.DestPath+".tmp_dw_"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
	if err != nil {
		log.Error("[download]", "Rename file", fileInfo.Name, "err", err)
	}
	_, err = f.client.Delete(pathInfo.SrcPath + ".lock_dw_" + fileInfo.Name)
	if err != nil {
		log.Error("[download]", "Delete file", fileInfo.Name, "err", err)
	}
}
func (f *FtpClient) upload(pathInfo *PathObject, fileInfo FtpFileInfo) {
	log.Info("[upload]", fileInfo.Name)

	err := os.Rename(pathInfo.SrcPath+fileInfo.Name, pathInfo.SrcPath+".lock_up_"+fileInfo.Name)
	if err != nil {
		log.Error("[upload]", "lock file", fileInfo.Name, "error", err)
		return
	}

	sf, err := os.Open(pathInfo.SrcPath + ".lock_up_" + fileInfo.Name)
	if err != nil {
		log.Error("[upload]", "Open file", fileInfo.Name, "error", err)
		return
	}
	defer sf.Close()

	err = f.client.UploadFile(pathInfo.DestPath+".tmp_up_"+fileInfo.Name,
		pathInfo.SrcPath+".lock_up_"+fileInfo.Name,
		false, nil)
	if err != nil {
		log.Error("[upload]", "file", fileInfo.Name, "err", err)
		return
	}

	_, err = f.client.Rename(pathInfo.DestPath+".tmp_up_"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
	if err != nil {
		log.Error("[upload]", "Rename file", fileInfo.Name, "err", err)
	}

	err = os.Remove(pathInfo.SrcPath + ".lock_up_" + fileInfo.Name)
	if err != nil {
		log.Error("[upload]", "Delete error", err)
	}
}

func FilterFile(files []string) []FtpFileInfo {
	fileList := make([]FtpFileInfo, 0)
	for i := range files {
		if strings.HasPrefix(files[i], "-") {
			infos := strings.Split(files[i], " ")
			for k := range infos {
				if checkDate(infos[k]) {
					fileInfo := FtpFileInfo{}
					fileInfo.Size = TypeCastTool.GetInt64(infos[k-1])
					fileInfo.Time, _ = time.ParseInLocation("0215:04", infos[k+1]+infos[k+2], time.Local)
					fileInfo.Name = getName(infos[k+3:])
					fileList = append(fileList, fileInfo)
					break
				}
			}
		}
	}
	return fileList
}
func getName(src []string) string {
	name := ""
	for i := range src {
		name += src[i] + " "
	}
	name = strings.TrimSuffix(name, " ")
	return name
}
func checkDate(date string) bool {
	switch date {
	case "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec":
		return true
	}
	return false
}
