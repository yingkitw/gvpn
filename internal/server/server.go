package server

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/yingkitw/gvpn/internal/auth"
	"github.com/yingkitw/gvpn/internal/config"
	"github.com/yingkitw/gvpn/internal/logger"
	"github.com/yingkitw/gvpn/internal/tunnel"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/net/ipv4"
)

const (
	HandshakeTimeout = 30 * time.Second
	KeepaliveInterval = 25 * time.Second
	PacketBufferSize = 2048
)

type Server struct {
	cfg         *config.Config
	tlsConfig   *tls.Config
	listener    net.Listener
	clients     map[string]*ClientInfo
	mu          sync.RWMutex
	tunnel      tunnel.PlatformTunnel
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	auth        *auth.AuthManager
	log         *logger.Logger
}

type ClientInfo struct {
	Client
	Username string
}

type Client struct {
	ID           string
	Conn         net.Conn
	VPNIP        net.IP
	RemoteAddr   net.Addr
	LastSeen     time.Time
	WriteQueue   chan []byte
	Key          []byte
	Cipher       cipher.AEAD
}

func New(cfg *config.Config, tlsConfig *tls.Config, authManager *auth.AuthManager, log *logger.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	
	var tun tunnel.PlatformTunnel
	var err error
	
	if tunnel.IsAvailable() {
		tun, err = tunnel.NewPlatformTunnel(cfg.VPNNetwork, cfg.MTU)
		if err != nil {
			log.Error("Failed to create tunnel: %v", err)
			tun = nil
		} else {
			log.Info("Tunnel interface created: %s", tun.Name())
		}
	} else {
		log.Info("Tunnel interface not available on this platform")
	}

	return &Server{
		cfg:       cfg,
		tlsConfig: tlsConfig,
		clients:   make(map[string]*ClientInfo),
		ctx:       ctx,
		cancel:    cancel,
		tunnel:    tun,
		auth:      authManager,
		log:       log,
	}
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	s.listener = listener

	if s.tunnel != nil {
		s.wg.Add(1)
		go s.handleTunnelTraffic()
	}

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *Server) Stop() {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for _, client := range s.clients {
		client.Conn.Close()
		close(client.WriteQueue)
	}
	s.clients = make(map[string]*ClientInfo)
	s.mu.Unlock()

	if s.tunnel != nil {
		s.tunnel.Close()
	}

	s.wg.Wait()
}

func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				s.log.Debug("Accept error: %v", err)
				continue
			}

			s.wg.Add(1)
			go s.handleClient(conn)
		}
	}
}

func (s *Server) handleClient(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	clientID := generateClientID()
	vpnIP := s.assignVPNIP()

	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		s.log.Error("Failed to generate encryption key: %v", err)
		return
	}

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		s.log.Error("Failed to create cipher: %v", err)
		return
	}

	client := &Client{
		ID:         clientID,
		Conn:       conn,
		VPNIP:      vpnIP,
		RemoteAddr: conn.RemoteAddr(),
		LastSeen:   time.Now(),
		WriteQueue: make(chan []byte, 100),
		Key:        key,
		Cipher:     cipher,
	}

	clientInfo := &ClientInfo{
		Client:   *client,
		Username: "anonymous",
	}

	s.mu.Lock()
	s.clients[clientID] = clientInfo
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, clientID)
		s.mu.Unlock()
		close(client.WriteQueue)
	}()

	s.log.Info("Client %s connected from %s, assigned VPN IP: %s", clientID, conn.RemoteAddr(), vpnIP)

	s.wg.Add(1)
	go s.handleClientWrites(client)

	buffer := make([]byte, PacketBufferSize)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(KeepaliveInterval * 2))
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}

			client.LastSeen = time.Now()
			if s.cfg.Debug {
				s.log.Debug("Received %d bytes from %s", n, clientID)
			}

			if err := s.handlePacket(client, buffer[:n]); err != nil {
				s.log.Debug("Packet handling error: %v", err)
			}
		}
	}
}

func (s *Server) handleClientWrites(client *Client) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case data, ok := <-client.WriteQueue:
			if !ok {
				return
			}

			client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := client.Conn.Write(data); err != nil {
				return
			}
		}
	}
}

func (s *Server) handlePacket(client *Client, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("empty packet")
	}

	packetType := data[0]
	payload := data[1:]

	switch packetType {
	case 0x01:
		return s.handleDataPacket(client, payload)
	case 0x02:
		return s.handleKeepalive(client)
	default:
		return fmt.Errorf("unknown packet type: %d", packetType)
	}
}

func (s *Server) handleDataPacket(client *Client, payload []byte) error {
	if len(payload) < 16 {
		return fmt.Errorf("payload too short")
	}

	nonce := payload[:12]
	ciphertext := payload[12:]
	
	plaintext, err := client.Cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("decryption failed: %v", err)
	}

	if s.tunnel != nil {
		if _, err := s.tunnel.Write(plaintext); err != nil && s.cfg.Debug {
			s.log.Debug("Tunnel write error: %v", err)
		}
	}

	return nil
}

func (s *Server) handleKeepalive(client *Client) error {
	client.LastSeen = time.Now()
	return nil
}

func (s *Server) handleTunnelTraffic() {
	defer s.wg.Done()

	if s.tunnel == nil {
		return
	}

	buffer := make([]byte, s.cfg.MTU)
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			n, err := s.tunnel.Read(buffer)
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				continue
			}

			s.routePacketToClients(buffer[:n])
		}
	}
}

func (s *Server) routePacketToClients(packet []byte) {
	header, err := ipv4.ParseHeader(packet)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, clientInfo := range s.clients {
		if clientInfo.VPNIP.Equal(header.Dst) {
			s.sendToClient(&clientInfo.Client, packet)
			return
		}
	}

	for _, clientInfo := range s.clients {
		s.sendToClient(&clientInfo.Client, packet)
	}
}

func (s *Server) sendToClient(client *Client, packet []byte) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		s.log.Debug("Failed to generate nonce: %v", err)
		return
	}

	ciphertext := client.Cipher.Seal(nil, nonce, packet, nil)
	
	payload := make([]byte, 1+12+len(ciphertext))
	payload[0] = 0x01
	copy(payload[1:], nonce)
	copy(payload[13:], ciphertext)

	select {
	case client.WriteQueue <- payload:
	default:
		s.log.Debug("Write queue full for client %s", client.ID)
	}
}

func (s *Server) assignVPNIP() net.IP {
	ip, ipnet, _ := net.ParseCIDR(s.cfg.VPNNetwork)
	
	s.mu.Lock()
	defer s.mu.Unlock()

	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		available := true
		for _, clientInfo := range s.clients {
			if clientInfo.VPNIP.Equal(ip) {
				available = false
				break
			}
		}
		if available {
			result := make(net.IP, len(ip))
			copy(result, ip)
			return result
		}
	}

	return nil
}

func generateClientID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func (s *Server) GetClients() map[string]*ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	clients := make(map[string]*ClientInfo)
	for id, client := range s.clients {
		clients[id] = client
	}
	return clients
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}