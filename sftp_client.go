package main

import (
	"context"
	"fmt"
	"github.com/pkg/sftp"
	log "github.com/wuzuoliang/micro-backend/MyLogger"
	"github.com/wuzuoliang/micro-backend/Tools/StructTool"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SFtpClient struct {
	AutoFtpInterface
	client *sftp.Client
}

func (s *SFtpClient) connect(info *FtpInfo, ctx context.Context) error {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(info.Passwd))

	clientConfig = &ssh.ClientConfig{
		User:    info.User,
		Auth:    auth,
		Timeout: 30 * time.Second,
		// Need to verify the server, do not verify to return nil, click HostKeyCallback to see the source code will know
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", info.Ip, info.Port)
	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return err
	}

	// create sftp client
	if s.client, err = sftp.NewClient(sshClient); err != nil {
		return err
	}
	log.Info("[SFTP connect] success", "PATH_ID", ctx.Value("PATH_ID"))

	return nil
}
func (s *SFtpClient) handle(pathInfo *PathObject) {
	defer s.close()
	pathInfo.fileTransC = make(chan FtpFileInfo, 1024*pathInfo.MultiGo)
	pathInfo.sets = StructTool.NewSet()
	r, _ := regexp.Compile(pathInfo.CompareStr)
	if pathInfo.TransMode == TRANS_TYPE_DOWNLOAD {
		s.scanFileFromRemotePath(pathInfo, r)
	} else if pathInfo.TransMode == TRANS_TYPE_UPLOAD {
		s.scanFileFromLocalPath(pathInfo, r)
	} else {
		log.Error("[handle]", "no such trans_mode", pathInfo.TransMode)
	}
}

func (s *SFtpClient) close() error {
	if s.client != nil {
		err := s.client.Close()
		if err != nil {
			log.Info("[SFTP close] fail", "error", err)
			return err
		}
		log.Info("[SFTP close] success")
		return err
	} else {
		log.Info("[SFTP close] no need close")
		return nil
	}
}

func (s *SFtpClient) consumer(pathInfo *PathObject) {
	for {
		select {
		case fileInfo := <-pathInfo.fileTransC:
			if pathInfo.TransMode == TRANS_TYPE_DOWNLOAD {
				s.download(pathInfo, fileInfo)
			} else if pathInfo.TransMode == TRANS_TYPE_UPLOAD {
				s.upload(pathInfo, fileInfo)
			}
		}
	}
}

func (s *SFtpClient) scanFileFromRemotePath(pathInfo *PathObject, r *regexp.Regexp) {
	for i := 0; i < pathInfo.MultiGo; i++ {
		go s.consumer(pathInfo)
	}
	for {
		fs := s.client.Walk(pathInfo.SrcPath)
		for fs.Step() {
			if fs.Err() != nil {
				continue
			}
			if fs.Stat().IsDir() {
				continue
			}
			fileName := strings.Replace(fs.Path(), pathInfo.SrcPath, "", -1)
			if strings.HasPrefix(fileName, ".") {
				continue
			}
			fileInfo := FtpFileInfo{fs.Stat().Name(), fs.Stat().Size(), fs.Stat().ModTime()}
			hashKey := MakeHashKey(fileInfo)
			switch pathInfo.CompareMode {
			case COMPARE_ALL:
				if !pathInfo.sets.Exist(fileName) {
					pathInfo.fileTransC <- fileInfo
					pathInfo.sets.Insert(hashKey)
				}
			case COMPARE_PRE:
				if strings.HasPrefix(fileName, pathInfo.CompareStr) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			case COMPARE_SUFF:
				if strings.HasSuffix(fileName, pathInfo.CompareStr) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			case COMPARE_REG:
				if r.MatchString(fileName) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			}
		}
		log.Debug("[handle]", "sleep")
		time.Sleep(time.Minute)
	}
}

func (s *SFtpClient) scanFileFromLocalPath(pathInfo *PathObject, r *regexp.Regexp) {
	for i := 0; i < pathInfo.MultiGo; i++ {
		go s.consumer(pathInfo)
	}
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
			hashKey := MakeHashKey(fileInfo)
			switch pathInfo.CompareMode {
			case COMPARE_ALL:
				if !pathInfo.sets.Exist(fileName) {
					pathInfo.fileTransC <- fileInfo
					pathInfo.sets.Insert(hashKey)
				}
			case COMPARE_PRE:
				if strings.HasPrefix(fileName, pathInfo.CompareStr) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			case COMPARE_SUFF:
				if strings.HasSuffix(fileName, pathInfo.CompareStr) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			case COMPARE_REG:
				if r.MatchString(fileName) {
					if !pathInfo.sets.Exist(fileName) {
						pathInfo.fileTransC <- fileInfo
						pathInfo.sets.Insert(hashKey)
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Error("[handle]", "scan local file error", err)
		}
		log.Debug("[handle]", "sleep")
		time.Sleep(time.Minute)
	}
}

func (s *SFtpClient) download(pathInfo *PathObject, fileInfo FtpFileInfo) {

	defer pathInfo.sets.Remove(MakeHashKey(fileInfo))

	err := s.client.Rename(pathInfo.SrcPath+fileInfo.Name, pathInfo.SrcPath+".lock_dw"+fileInfo.Name)
	if err != nil {
		log.Error("[download]", "Rename err", err)
		return
	}

	if fileInfo.Size <= MULTI_GO_FILE_SIZE {
		srcFile, err := s.client.Open(pathInfo.SrcPath + ".lock_dw" + fileInfo.Name)
		if err != nil {
			log.Error("[download]", "Open src err", err)
			return
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(pathInfo.DestPath+".tmp_dw"+fileInfo.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
		if err != nil {
			log.Error("[download]", "Open dest err", err)
			return
		}
		defer destFile.Close()

		err = CopyFileSingleDown(srcFile, destFile)
		if err != nil {
			log.Error("[download]", "CopyFileSingle err", err)
			return
		}

		err = os.Rename(pathInfo.DestPath+".tmp_dw"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
		if err != nil {
			log.Error("[download]", "Merge part error", err)
		}

		err = s.client.Remove(pathInfo.SrcPath + ".lock_dw" + fileInfo.Name)
		if err != nil {
			log.Error("[download]", "Remove error", err)
			return
		}
		log.Info("[download]", "file", fileInfo.Name, "Success")
	} else {
		allSucc := true
		partFlag := make([]bool, pathInfo.MultiThread)
		partSize := fileInfo.Size / int64(pathInfo.MultiThread)
		partLeft := fileInfo.Size % int64(pathInfo.MultiThread)
		log.Info("[download]", "fileSize", fileInfo.Size, "partSize", partSize, "final part need addition size", partLeft)
		wg := sync.WaitGroup{}
		wg.Add(pathInfo.MultiThread)
		for i := 0; i < pathInfo.MultiThread; i++ {
			go func(i int) {
				defer func(err error) {
					if err != nil {
						partFlag[i] = false
						allSucc = false
					} else {
						partFlag[i] = true
					}
					log.Debug("[download]", fileInfo.Name, "part", i, "finish")
					wg.Done()
				}(err)
				destFile, err := os.OpenFile(pathInfo.DestPath+".tmp_dw"+fileInfo.Name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0744)
				if err != nil {
					log.Error("[download]", "Open dest err", err)
					return
				}
				defer destFile.Close()

				srcFile, err := s.client.Open(pathInfo.SrcPath + ".lock_dw" + fileInfo.Name)
				if err != nil {
					log.Error("[download]", "Open src err", err)
					return
				}
				defer srcFile.Close()

				err = CopyFileMultiGoDown(srcFile, destFile, i, partSize, partLeft, i == pathInfo.MultiThread-1)
			}(i)
		}
		wg.Wait()
		if allSucc {
			log.Info("[download]", "Success file ", fileInfo.Name)
			err = os.Rename(pathInfo.DestPath+".tmp_"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
			if err != nil {
				log.Error("[download]", "Merge part error", err)
			}
			err = s.client.Remove(pathInfo.SrcPath + ".lock_dw" + fileInfo.Name)
			if err != nil {
				log.Error("[download]", "Remove error", err)
				return
			}
		} else {
			log.Error("[download]", "Detail info:", func(b []bool) string {
				var ret string
				for k, v := range b {
					if v {
						ret += "[Part " + strconv.Itoa(k) + ": Succ] "
					} else {
						ret += "[Part " + strconv.Itoa(k) + ": Fail] "
					}
				}
				return ret
			}(partFlag))
		}
	}
}

func (s *SFtpClient) upload(pathInfo *PathObject, fileInfo FtpFileInfo) {
	log.Debug("[upload]", "filename", fileInfo.Name)

	defer pathInfo.sets.Remove(MakeHashKey(fileInfo))

	err := os.Rename(pathInfo.SrcPath+fileInfo.Name, pathInfo.SrcPath+".lock_up"+fileInfo.Name)
	if err != nil {
		log.Error("[upload]", "Rename err", err)
		return
	}

	if fileInfo.Size <= MULTI_GO_FILE_SIZE {
		srcFile, err := os.Open(pathInfo.SrcPath + ".lock_up" + fileInfo.Name)
		if err != nil {
			log.Error("[upload]", "Open src err", err)
			return
		}
		defer srcFile.Close()

		destFile, err := s.client.Create(pathInfo.DestPath + ".tmp_up" + fileInfo.Name)
		if err != nil {
			log.Error("[upload]", "Open dest err", err)
			return
		}
		defer destFile.Close()

		err = CopyFileSingleUp(srcFile, destFile, fileInfo.Size)
		if err != nil {
			log.Error("[upload]", "CopyFileSingle err", err)
			return
		}

		err = s.client.Rename(pathInfo.DestPath+".tmp_up"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
		if err != nil {
			log.Error("[upload]", "Merge part error", err)
		}

		err = os.Remove(pathInfo.SrcPath + ".lock_up" + fileInfo.Name)
		if err != nil {
			log.Error("[upload]", "Remove error", err)
			return
		}
		log.Info("[upload]", "file", fileInfo.Name, "Success")

	} else {
		allSucc := true
		partFlag := make([]bool, pathInfo.MultiThread)
		partSize := fileInfo.Size / int64(pathInfo.MultiThread)
		partLeft := fileInfo.Size % int64(pathInfo.MultiThread)
		log.Info("[upload]", "fileSize", fileInfo.Size, "partSize", partSize, "final part need addition size", partLeft)
		wg := sync.WaitGroup{}
		wg.Add(pathInfo.MultiThread)
		for i := 0; i < pathInfo.MultiThread; i++ {
			go func(i int) {
				defer func(err error) {
					if err != nil {
						partFlag[i] = false
						allSucc = false
					} else {
						partFlag[i] = true
					}
					log.Debug("[upload]", fileInfo.Name, "part", i, "finish")
					wg.Done()
				}(err)
				destFile, err := s.client.OpenFile(pathInfo.DestPath+".tmp_up"+fileInfo.Name, os.O_RDWR|os.O_CREATE|os.O_APPEND)
				if err != nil {
					log.Error("[upload]", "Open dest err", err)
					return
				}
				defer destFile.Close()

				srcFile, err := os.Open(pathInfo.SrcPath + ".lock_up" + fileInfo.Name)
				if err != nil {
					log.Error("[upload]", "Open src err", err)
					return
				}
				defer srcFile.Close()

				err = CopyFileMultiGoUp(srcFile, destFile, i, partSize, partLeft, i == pathInfo.MultiThread-1)
			}(i)
		}
		wg.Wait()
		if allSucc {
			log.Info("[upload]", "Success file ", fileInfo.Name)
			err = s.client.Rename(pathInfo.DestPath+".tmp_up"+fileInfo.Name, pathInfo.DestPath+fileInfo.Name)
			if err != nil {
				log.Error("[upload]", "Merge part error", err)
			}
			err = os.Remove(pathInfo.SrcPath + ".lock_up" + fileInfo.Name)
			if err != nil {
				log.Error("[upload]", "Remove error", err)
				return
			}
		} else {
			log.Error("[upload]", "Detail info:", func(b []bool) string {
				var ret string
				for k, v := range b {
					if v {
						ret += "[Part " + strconv.Itoa(k) + ": Succ] "
					} else {
						ret += "[Part " + strconv.Itoa(k) + ": Fail] "
					}
				}
				return ret
			}(partFlag))
		}
	}
}
