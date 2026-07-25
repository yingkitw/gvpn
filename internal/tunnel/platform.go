package tunnel

import (
	"fmt"
	"os"
	"runtime"
)

const (
	TUNSETIFF = 0x400454ca
	IFF_TUN   = 0x0001
	IFF_TAP   = 0x0002
	IFF_NO_PI = 0x1000
)

type PlatformTunnel interface {
	Close() error
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Name() string
	Configure(network string) error
}

func NewPlatformTunnel(network string, mtu int) (PlatformTunnel, error) {
	switch runtime.GOOS {
	case "linux":
		return NewLinuxTunnel(network, mtu)
	case "darwin":
		return NewDarwinTunnel(network, mtu)
	case "windows":
		return NewWindowsTunnel(network, mtu)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func IsAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := os.Stat("/dev/net/tun")
		return err == nil
	case "darwin":
		return true
	case "windows":
		return true
	default:
		return false
	}
}