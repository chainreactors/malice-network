package utils

import (
	"crypto/md5"
	"fmt"
	"os"
	"runtime"
	"strings"
)

func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// GetMachineID 获取机器唯一标识符
func GetMachineID() string {
	var machineInfo []string

	// 获取主机名
	if hostname, err := os.Hostname(); err == nil {
		machineInfo = append(machineInfo, hostname)
	}

	// 获取操作系统信息
	machineInfo = append(machineInfo, runtime.GOOS)
	machineInfo = append(machineInfo, runtime.GOARCH)

	// 获取环境变量中的一些唯一标识符
	if envVars := []string{"COMPUTERNAME", "HOSTNAME", "USERNAME", "USER"}; len(envVars) > 0 {
		for _, envVar := range envVars {
			if value := os.Getenv(envVar); value != "" {
				machineInfo = append(machineInfo, value)
			}
		}
	}

	// 组合所有信息并生成MD5哈希
	combined := strings.Join(machineInfo, "_")
	hash := md5.Sum([]byte(combined))

	// 返回前16个字符作为机器码
	return fmt.Sprintf("%x", hash)[:16]
}

func FirstOrEmpty(arr []string) string {
	if len(arr) > 0 {
		return arr[0]
	}
	return ""
}
