package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	// "time"
)

type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

type ParameterStore struct {
	namespaces map[string]map[string]interface{}
	clients    map[string][]net.Addr
	mutex      sync.RWMutex
}

type Server struct {
	config *Config
	store  *ParameterStore
	conn   *net.UDPConn
}

func NewParameterStore() *ParameterStore {
	return &ParameterStore{
		namespaces: make(map[string]map[string]interface{}),
		clients:    make(map[string][]net.Addr),
	}
}

func (ps *ParameterStore) SetParameter(namespace, key string, value interface{}, clientAddr net.Addr) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	if ps.namespaces[namespace] == nil {
		ps.namespaces[namespace] = make(map[string]interface{})
	}
	
	ps.namespaces[namespace][key] = value
	
	// Add client to namespace if not already present
	found := false
	for _, addr := range ps.clients[namespace] {
		if addr.String() == clientAddr.String() {
			found = true
			break
		}
	}
	if !found {
		ps.clients[namespace] = append(ps.clients[namespace], clientAddr)
	}
}

func (ps *ParameterStore) GetParameter(namespace, key string) (interface{}, bool) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	if params, exists := ps.namespaces[namespace]; exists {
		if value, exists := params[key]; exists {
			return value, true
		}
	}
	return nil, false
}

func (ps *ParameterStore) GetAllParameters(namespace string) map[string]interface{} {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()
	
	if params, exists := ps.namespaces[namespace]; exists {
		result := make(map[string]interface{})
		for k, v := range params {
			result[k] = v
		}
		return result
	}
	return make(map[string]interface{})
}

func (ps *ParameterStore) AddClient(namespace string, clientAddr net.Addr) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	
	if ps.clients[namespace] == nil {
		ps.clients[namespace] = make([]net.Addr, 0)
	}
	
	// Check if client already exists
	for _, addr := range ps.clients[namespace] {
		if addr.String() == clientAddr.String() {
			return
		}
	}
	
	ps.clients[namespace] = append(ps.clients[namespace], clientAddr)
}

func (ps *ParameterStore) BroadcastToNamespace(namespace string, data []byte, conn *net.UDPConn, sender net.Addr) {
	ps.mutex.RLock()
	clients := make([]net.Addr, len(ps.clients[namespace]))
	copy(clients, ps.clients[namespace])
	ps.mutex.RUnlock()
	
	for _, clientAddr := range clients {
		// Don't send back to sender
		if clientAddr.String() != sender.String() {
			_, err := conn.WriteToUDP(data, clientAddr.(*net.UDPAddr))
			if err != nil {
				log.Printf("Failed to send to client %s: %v", clientAddr, err)
			}
		}
	}
}

func NewServer(config *Config) (*Server, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		return nil, err
	}
	
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	
	return &Server{
		config: config,
		store:  NewParameterStore(),
		conn:   conn,
	}, nil
}

func (s *Server) handleMessage(data []byte, clientAddr net.Addr) {
	var message map[string]interface{}
	if err := json.Unmarshal(data, &message); err != nil {
		log.Printf("Failed to parse JSON from %s: %v", clientAddr, err)
		return
	}
	
	// Handle namespace registration
	if namespace, exists := message["__namespace__"]; exists {
		namespaceStr := namespace.(string)
		s.store.AddClient(namespaceStr, clientAddr)
		log.Printf("Client %s registered to namespace '%s'", clientAddr, namespaceStr)
		
		// Send all existing parameters in this namespace to the new client
		params := s.store.GetAllParameters(namespaceStr)
		if len(params) > 0 {
			response, _ := json.Marshal(params)
			s.conn.WriteToUDP(response, clientAddr.(*net.UDPAddr))
		}
		return
	}
	
	// Handle parameter updates
	// Find which namespace this client belongs to
	var clientNamespace string
	s.store.mutex.RLock()
	for namespace, clients := range s.store.clients {
		for _, addr := range clients {
			if addr.String() == clientAddr.String() {
				clientNamespace = namespace
				break
			}
		}
		if clientNamespace != "" {
			break
		}
	}
	s.store.mutex.RUnlock()
	
	if clientNamespace == "" {
		log.Printf("Client %s not registered to any namespace", clientAddr)
		return
	}
	
	// Update parameters and broadcast to other clients in the same namespace
	for key, value := range message {
		if key != "__namespace__" {
			s.store.SetParameter(clientNamespace, key, value, clientAddr)
			log.Printf("Updated parameter %s=%v in namespace '%s' from %s", key, value, clientNamespace, clientAddr)
		}
	}
	
	// Broadcast the update to other clients in the same namespace
	s.store.BroadcastToNamespace(clientNamespace, data, s.conn, clientAddr)
}

func (s *Server) Run() {
	log.Printf("ConParamLive backend server starting on %s:%d", s.config.Host, s.config.Port)
	
	buffer := make([]byte, 1024)
	
	for {
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP message: %v", err)
			continue
		}
		
		go s.handleMessage(buffer[:n], clientAddr)
	}
}

func (s *Server) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	
	return &config, nil
}

func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	server.Run()
}