package tunnel

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

type WindowsTunnel struct {
	name   string
	mtu    int
	closed bool
}

func NewWindowsTunnel(network string, mtu int) (*WindowsTunnel, error) {
	if _, err := exec.LookPath("netsh"); err != nil {
		return nil, fmt.Errorf("netsh command not found: %v", err)
	}

	tun := &WindowsTunnel{
		name:   "gvpn0",
		mtu:    mtu,
		closed: false,
	}

	if err := tun.Configure(network); err != nil {
		tun.Close()
		return nil, fmt.Errorf("failed to configure TUN interface: %v", err)
	}

	return tun, nil
}

func (t *WindowsTunnel) Configure(network string) error {
	ip, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("invalid network: %v", err)
	}

	netmask := fmt.Sprintf("%d.%d.%d.%d", ipnet.Mask[0], ipnet.Mask[1], ipnet.Mask[2], ipnet.Mask[3])

	commands := [][]string{
		{"netsh", "interface", "ipv4", "add", "address", t.name, ip.String(), netmask},
		{"netsh", "interface", "set", "interface", t.name, "enabled"},
	}

	for _, cmd := range commands {
		output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
		if err != nil {
			if strings.Contains(string(output), "not found") {
				continue
			}
			return fmt.Errorf("failed to execute %v: %v (output: %s)", cmd, err, string(output))
		}
	}

	return nil
}

func (t *WindowsTunnel) Read(buf []byte) (int, error) {
	return 0, fmt.Errorf("not implemented on windows")
}

func (t *WindowsTunnel) Write(buf []byte) (int, error) {
	return 0, fmt.Errorf("not implemented on windows")
}

func (t *WindowsTunnel) Close() error {
	if t.closed {
		return nil
	}

	t.closed = true
	exec.Command("netsh", "interface", "set", "interface", t.name, "disabled").Run()
	return nil
}

func (t *WindowsTunnel) Name() string {
	return t.name
}