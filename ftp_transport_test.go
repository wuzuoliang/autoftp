package main

import (
	"testing"
	"time"
)

func TestMakeHashKey(t *testing.T) {
	t.Log(MakeHashKey(FtpFileInfo{"sadased.txt", 1, time.Now()}))
}
