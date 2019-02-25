package main

import (
	"crypto/sha1"
	"errors"
	"github.com/pkg/sftp"
	log "github.com/wuzuoliang/micro-backend/MyLogger"
	"github.com/wuzuoliang/micro-backend/Tools/TypeCastTool"
	"os"
	"strconv"
)

func MakeHashKey(fileInfo FtpFileInfo) string {
	h := sha1.New()
	h.Write(TypeCastTool.StringToBytes(fileInfo.Name + strconv.FormatInt(fileInfo.Size, 10) + fileInfo.Time.Format("20060102150405")))
	bs := h.Sum(nil)
	return TypeCastTool.BytesToString(bs)
}

/*
This follow functions are used by sftp_client
*/
func CopyFileSingleDown(src *sftp.File, dest *os.File) error {
	bn, err := src.WriteTo(dest)
	if err != nil {
		log.Error("[CopyFileSingleDown]", err)
		return err
	}
	log.Debug("[CopyFileSingleDown]", "file", dest.Name(), "size", bn)
	return nil
}

func CopyFileSingleUp(src *os.File, dest *sftp.File, fileSize int64) error {
	buf := make([]byte, fileSize)
	rLen, err := src.Read(buf)
	if err != nil {
		log.Error("[CopyFileSingleUp]", "Read error", err)
		return err
	}
	if int64(rLen) != fileSize {
		log.Error("[CopyFileSingleUp]", "Read file error,want read", fileSize, "but read ", rLen)
		return errors.New("not read all")
	}

	bn, err := dest.Write(buf)
	if err != nil {
		log.Error("[CopyFileSingleUp]", err)
		return err
	}
	log.Debug("[CopyFileSingleUp]", "file", dest.Name(), "size", bn)
	return nil
}

func CopyFileMultiGoDown(src *sftp.File, dest *os.File, partIndex int, partSize, partLeft int64, finalPart bool) error {
	buf := make([]byte, partSize)
	if finalPart {
		log.Debug("[CopyFileMultiGoDown]", "this is final part,reassign buf size", len(buf))
		buf = make([]byte, partSize+partLeft)
	}
	seek := int64(partIndex) * partSize
	ofSet, err := src.Seek(seek, 0)
	if err != nil {
		log.Error("[CopyFileMultiGoDown]", dest.Name(), "Seek error", err)
		return err
	}
	if ofSet != seek {
		log.Error("[CopyFileMultiGoDown]", dest.Name(), "ofSet error", ofSet, "wanted", seek)
		return errors.New("ofSet error")
	}
	rLen, err := src.Read(buf)
	if err != nil {
		log.Error("[CopyFileMultiGoDown]", dest.Name(), "Read error", err)
		return err
	}
	log.Debug("[CopyFileMultiGoDown]", dest.Name(), "Part", partIndex, "Read length", rLen)

	wLen, err := dest.WriteAt(buf, seek)
	if err != nil {
		log.Error("[CopyFileMultiGoDown]", dest.Name(), "Write error", err)
		return err
	}
	log.Debug("[CopyFileMultiGoDown]", dest.Name(), "Part", partIndex, "Write length", wLen)

	if rLen != wLen {
		log.Error("[CopyFileMultiGoDown]", "Read or Write error")
		return errors.New("Read or Write error")
	}
	return nil
}

func CopyFileMultiGoUp(src *os.File, dest *sftp.File, partIndex int, partSize, partLeft int64, finalPart bool) error {
	buf := make([]byte, partSize)
	if finalPart {
		log.Debug("[CopyFileMultiGoUp]", "this is final part,reassign buf size", len(buf))
		buf = make([]byte, partSize+partLeft)
	}
	seek := int64(partIndex) * partSize
	srcOfSet, err := src.Seek(seek, 0)
	if err != nil {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "Seek error", err)
		return err
	}
	if srcOfSet != seek {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "srcOfSet error", srcOfSet, "wanted", seek)
		return errors.New("ofSet error")
	}
	rLen, err := src.Read(buf)
	if err != nil {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "Read error", err)
		return err
	}
	log.Debug("[CopyFileMultiGoUp]", dest.Name(), "Part", partIndex, "Read length", rLen)

	destOfSet, err := dest.Seek(seek, 0)
	if err != nil {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "Seek error", err)
		return err
	}
	if destOfSet != seek {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "destOfSet error", destOfSet, "wanted", seek)
		return errors.New("ofSet error")
	}
	wLen, err := dest.Write(buf)
	if err != nil {
		log.Error("[CopyFileMultiGoUp]", dest.Name(), "Write error", err)
		return err
	}
	log.Debug("[CopyFileMultiGoUp]", dest.Name(), "Part", partIndex, "Write length", wLen)

	if rLen != wLen {
		log.Error("[CopyFileMultiGoUp]", "Read or Write error")
		return errors.New("Read or Write error")
	}
	return nil
}
