package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yingkitw/gvpn/internal/auth"
	"github.com/yingkitw/gvpn/internal/config"
	"github.com/yingkitw/gvpn/internal/logger"
	"github.com/yingkitw/gvpn/internal/routing"
	"github.com/yingkitw/gvpn/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log := logger.Init(cfg.Debug)
	log.Info("Starting GVPN Server...")

	certPEM, keyPEM, err := generateSelfSignedCert()
	if err != nil {
		log.Error("Failed to generate certificate: %v", err)
		os.Exit(1)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Error("Failed to create certificate pair: %v", err)
		os.Exit(1)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	authManager := auth.New(24 * time.Hour)
	authManager.AddUser("admin", "admin123")
	authManager.StartCleanupTask(1 * time.Hour)

	vpnServer := server.New(cfg, tlsConfig, authManager, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := vpnServer.Start(ctx); err != nil {
			log.Error("Server error: %v", err)
		}
	}()

	log.Info("VPN server started on %s", cfg.ListenAddr)
	log.Info("VPN network: %s", cfg.VPNNetwork)
	log.Info("Default user: admin/admin123")
	log.Info("Press Ctrl+C to stop")

	go handleCommands(vpnServer, log)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutting down server...")
	vpnServer.Stop()
	log.Info("Server stopped")
}

func generateSelfSignedCert() ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"GVPN"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	return certPEM, keyPEM, nil
}

func handleCommands(srv *server.Server, log *logger.Logger) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		switch cmd {
		case "clients":
			clients := srv.GetClients()
			log.Info("Connected clients: %d", len(clients))
			for id, client := range clients {
				log.Info("  %s: %s -> %s", id, client.RemoteAddr, client.VPNIP)
			}
		case "routes":
			routingMgr := routing.New("10.8.0.0/24", "10.8.0.1")
			routes, err := routingMgr.GetRoutingTable()
			if err != nil {
				log.Error("Failed to get routes: %v", err)
			} else {
				log.Info("Routing table:")
				for _, route := range routes {
					log.Info("  %s", route)
				}
			}
		case "help":
			log.Info("Available commands: clients, routes, help, quit")
		case "quit", "exit":
			log.Info("Shutting down...")
			srv.Stop()
			os.Exit(0)
		default:
			log.Info("Unknown command: %s", cmd)
		}
	}
}