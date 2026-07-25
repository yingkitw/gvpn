package main

import (
	"crypto/cipher"
	"crypto/rand"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

const (
	ServerAddr     = "localhost:1194"
	KeepaliveInterval = 25 * time.Second
)

func main() {
	log.Println("GVPN Client")

	conn, err := net.Dial("tcp", ServerAddr)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	log.Printf("Connected to %s", ServerAddr)

	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}

	cipher, err := chacha20poly1305.NewX(key)
	if err != nil {
		log.Fatalf("Failed to create cipher: %v", err)
	}

	done := make(chan struct{})
	go sendKeepalive(conn, done)
	go receiveData(conn, cipher, done)

	<-done
	log.Println("Client disconnected")
}

func sendKeepalive(conn net.Conn, done chan struct{}) {
	ticker := time.NewTicker(KeepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			packet := []byte{0x02}
			if _, err := conn.Write(packet); err != nil {
				log.Printf("Failed to send keepalive: %v", err)
				close(done)
				return
			}
		case <-done:
			return
		}
	}
}

func receiveData(conn net.Conn, cipher cipher.AEAD, done chan struct{}) {
	buffer := make([]byte, 2048)
	
	for {
		conn.SetReadDeadline(time.Now().Add(KeepaliveInterval * 2))
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("Read error: %v", err)
			close(done)
			return
		}

		if n < 1 {
			continue
		}

		packetType := buffer[0]
		payload := buffer[1:n]

		switch packetType {
		case 0x01:
			handleDataPacket(payload, cipher)
		case 0x02:
			log.Printf("Unknown packet type: %d", packetType)
		}
	}
}

func handleDataPacket(payload []byte, cipher cipher.AEAD) {
	if len(payload) < 16 {
		log.Printf("Invalid payload length: %d", len(payload))
		return
	}

	nonce := payload[:12]
	ciphertext := payload[12:]

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Printf("Decryption failed: %v", err)
		return
	}

	if len(plaintext) >= 20 {
		headerLen := int((plaintext[0] & 0x0F)) * 4
		
		if len(plaintext) >= headerLen {
			srcIP := net.IP(plaintext[12:16])
			dstIP := net.IP(plaintext[16:20])
			protocol := plaintext[9]
			
			log.Printf("IP packet: %s -> %s, protocol: %d, length: %d", srcIP, dstIP, protocol, len(plaintext))
		}
	}
}