package main

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	// "time"
)

type Config struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	StoragePath string `json:"storage_path"`
}

type NamespaceStorage struct {
	Params      map[string]any `json:"params"`
	LastUpdated int64          `json:"last_updated"` // Unix 时间戳
	Clients     []net.UDPAddr  `json:"-"`            // 不存储在文件中
}

type Server struct {
	Storage map[string]NamespaceStorage
	Mutex   *sync.RWMutex
	Config  *Config
	Conn    *net.UDPConn
}

func main() {
	// 加载配置文件
	file, err := os.Open("config.json")
	if err != nil {
		return
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return
	}

	// 启动 UDP 服务器
	addr := net.UDPAddr{IP: net.ParseIP(config.Host), Port: config.Port}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return
	}
	defer conn.Close()

	// 创建服务器实例
	server := &Server{
		Storage: make(map[string]NamespaceStorage),
		Mutex:   &sync.RWMutex{},
		Config:  &config,
		Conn:    conn,
	}

	// 启动处理 goroutine
	// go server.handleConnections()

	// 阻塞主 goroutine
	select {}
}
