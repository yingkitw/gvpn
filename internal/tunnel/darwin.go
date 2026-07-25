package tunnel

import (
	"fmt"
	"net"
	"os/exec"
)

type DarwinTunnel struct {
	name   string
	mtu    int
	closed bool
}

func NewDarwinTunnel(network string, mtu int) (*DarwinTunnel, error) {
	if _, err := exec.LookPath("ifconfig"); err != nil {
		return nil, fmt.Errorf("ifconfig command not found: %v", err)
	}

	commands := [][]string{
		{"kextload", "-b", "net.sf.tuntaposx.tap"},
		{"ifconfig", "tap0", "create"},
		{"ifconfig", "tap0", network},
		{"ifconfig", "tap0", "up"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			continue
		}
	}

	tun := &DarwinTunnel{
		name:   "utun0",
		mtu:    mtu,
		closed: false,
	}

	if err := tun.Configure(network); err != nil {
		tun.Close()
		return nil, fmt.Errorf("failed to configure TUN interface: %v", err)
	}

	return tun, nil
}

func (t *DarwinTunnel) Configure(network string) error {
	ip, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("invalid network: %v", err)
	}

	commands := [][]string{
		{"ifconfig", t.name, ip.String(), ip.String(), "netmask", ipnet.Mask.String(), "up"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to execute %v: %v", cmd, err)
		}
	}

	return nil
}

func (t *DarwinTunnel) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("not implemented on darwin")
}

func (t *DarwinTunnel) Write(buf []byte) (int, error) {
	return 0, fmt.Errorf("not implemented on darwin")
}

func (t *DarwinTunnel) Close() error {
	if t.closed {
		return nil
	}

	t.closed = true
	exec.Command("ifconfig", t.name, "down").Run()
	return nil
}

func (t *DarwinTunnel) Name() string {
	return t.name
}