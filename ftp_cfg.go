package main

import (
	"github.com/wuzuoliang/micro-backend/Tools/StructTool"
	"time"
)

const (
	TYPE_FTP  = 0
	TYPE_SFTP = 1

	TRANS_TYPE_UPLOAD   = 0
	TRANS_TYPE_DOWNLOAD = 1

	COMPARE_ALL  = 0
	COMPARE_PRE  = 1
	COMPARE_SUFF = 2
	COMPARE_REG  = 3

	NOT_DELETE_FILE = 0
	DELETE_FILE     = 1

	MULTI_GO_FILE_SIZE = 1024 * 1024 * 10 // 大于这个数量开始多协程下载
)

type Cfg struct {
	Instances []Instance
}
type Instance struct {
	InstanceID int          // 实例ID
	FtpInfo    FtpInfo      // ftp配置信息
	FtpPaths   []PathObject // ftp采集信息
}
type FtpInfo struct {
	User      string
	Ip        string
	Port      int
	Passwd    string
	TransType int // 传输方式 0 ftp 1 sftp
}
type PathObject struct {
	MultiGo     int    // 任务多协程，SFTP支持，FTP不支持并发，默认1
	MultiThread int    // 单文件多线程，SFTP支持，FTP不支持并发，默认1
	PathID      int    // 路径ID
	TransMode   int    // 传输方向 0 上传 1 下载
	SrcPath     string // 源路径
	DestPath    string // 目标路径
	TmpPath     string // 临时路径  考虑可以先把处理的文件先搬到临时路径下，再进行上传或处理操作 （待完善）

	IsBak   int    // 是否生成备份文件 0 不备份 1 备份 （待完善）
	BakPath string // 备份路径

	IsDel int // 传输完成是否删除源文件 0 不删除 1 删除 （待完善）

	CompareMode int    // 匹配模式 0 全匹配 1 前缀匹配 2 后缀匹配 3 正则匹配
	CompareStr  string // 匹配规则串
	SleepTime   int    // 传输结束休眠时间

	sets       *StructTool.Set
	fileTransC chan FtpFileInfo
}
type FtpFileInfo struct {
	Name string
	Size int64
	Time time.Time
}
