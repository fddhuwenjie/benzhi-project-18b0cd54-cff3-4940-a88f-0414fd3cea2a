package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

type config struct {
	address   string
	dataFile  string
	selfCheck bool
}

func parseConfig(args []string) (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if raw := os.Getenv("PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil {
			return config{}, fmt.Errorf("PORT 必须是端口号: %w", err)
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	fs := flag.NewFlagSet("anaerobic-release", flag.ContinueOnError)
	addr := fs.String("addr", defaultAddr, "HTTP 监听地址")
	data := fs.String("data", filepath.Join("data", "snapshot.json"), "JSON 快照路径")
	self := fs.Bool("self-check", false, "运行真实回环业务自检后退出")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 0 {
		return config{}, errors.New("存在未识别的位置参数")
	}
	if err := validateAddress(*addr); err != nil {
		return config{}, err
	}
	return config{address: *addr, dataFile: *data, selfCheck: *self}, nil
}

func validateAddress(addr string) error {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("-addr 格式无效: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("监听地址必须使用明确的回环 IP")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1024 || port > 65535 {
		return errors.New("监听端口必须在 1024 到 65535 之间")
	}
	return nil
}
