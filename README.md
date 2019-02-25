# autoftp

自动扫描目录下指定类型文件,根据配置规则实现上传和下载。

**功能描述**
支持ftp，sftp两种方式的上传和下载  
ftp暂时只支持单通道轮询处理  
sftp支持多协程并发处理  

**配置文件样例**
```json
{
  "Instances": [
    {
      "InstanceID": 1,
      "FtpInfo": {
        "User": "wuzl",
        "Ip": "127.0.0.1",
        "Port": 21,
        "Passwd": "",
        "TransType": 0 
      },
      "FtpPaths": [
        {
          "PathID": 1,
          "TransMode": 1,
          "MultiGo": 1,
          "MultiThread": 1,
          "SrcPath": "/home/wuzl/ftp/src/",
          "DestPath": "/Users/wuzuoliang/Downloads/dwtest/",
          "TmpPath": "",
          "BakPath": "",
          "IsBak": 0,
          "IsDel": 0,
          "CompareMode": 0,
          "CompareStr": "*",
          "SleepTime": 1
        },
        {
          "PathID": 2,
          "TransMode": 0,
          "MultiGo": 1,
          "MultiThread": 1,
          "SrcPath": "/Users/wuzuoliang/Downloads/uptest/",
          "DestPath": "/home/wuzl/ftp/dest/",
          "TmpPath": "",
          "BakPath": "",
          "IsBak": 0,
          "IsDel": 0,
          "CompareMode": 0,
          "CompareStr": "*",
          "SleepTime": 1
        }
      ]
    },
    {
      "InstanceID": 2,
      "FtpInfo": {
        "User": "dev",
        "Ip": "192.168.199.222",
        "Port": 22,
        "Passwd": "Ln2AIFoT",
        "TransType": 1
      },
      "FtpPaths": [
        {
          "PathID": 1,
          "TransMode": 0,
          "MultiGo": 5,
          "MultiThread": 10,
          "SrcPath": "/Users/wuzuoliang/Downloads/uptest/",
          "DestPath": "/home/dev/baker/upload/",
          "TmpPath": "",
          "BakPath": "",
          "IsBak": 0,
          "IsDel": 0,
          "CompareMode": 0,
          "CompareStr": "*",
          "SleepTime": 1
        },
        {
          "PathID": 2,
          "TransMode": 1,
          "MultiGo": 2,
          "MultiThread": 3,
          "SrcPath": "/home/dev/baker/src/",
          "DestPath": "/Users/wuzuoliang/Downloads/dwtest/",
          "TmpPath": "",
          "BakPath": "",
          "IsBak": 0,
          "IsDel": 0,
          "CompareMode": 3,
          "CompareStr": ".log",
          "SleepTime": 2
        }
      ]
    }
  ]
}
```

**配置文件结构信息**
```go
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
}
```
**使用方法** 
```
go build
./autoftp -i 1 -c /etc/autoftp/auto_ftp.json
```