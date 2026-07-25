package tunnel

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

type LinuxTunnel struct {
	fd     int
	mtu    int
	name   string
	closed bool
}

type ifReq struct {
	Name  [16]byte
	Flags uint16
	pad   [14]byte
}

func NewLinuxTunnel(network string, mtu int) (*LinuxTunnel, error) {
	if _, err := exec.LookPath("ip"); err != nil {
		return nil, fmt.Errorf("ip command not found: %v", err)
	}

	fd, err := syscall.Open("/dev/net/tun", syscall.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open TUN device: %v", err)
	}

	var ifr ifReq
	copy(ifr.Name[:], "tun%d")
	ifr.Flags = IFF_TUN | IFF_NO_PI

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), TUNSETIFF, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to create TUN interface: %v", errno)
	}

	tun := &LinuxTunnel{
		fd:     fd,
		mtu:    mtu,
		name:   string(ifr.Name[:]),
		closed: false,
	}

	if err := tun.Configure(network); err != nil {
		tun.Close()
		return nil, fmt.Errorf("failed to configure TUN interface: %v", err)
	}

	return tun, nil
}

func (t *LinuxTunnel) Configure(network string) error {
	commands := [][]string{
		{"ip", "addr", "add", network, "dev", t.name},
		{"ip", "link", "set", "dev", t.name, "up"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to execute %v: %v", cmd, err)
		}
	}

	return nil
}

func (t *LinuxTunnel) Read(buf []byte) (int, error) {
	if t.closed {
		return 0, fmt.Errorf("tunnel closed")
	}

	n, err := syscall.Read(t.fd, buf)
	if err != nil {
		return 0, fmt.Errorf("read error: %v", err)
	}

	return n, nil
}

func (t *LinuxTunnel) Write(buf []byte) (int, error) {
	if t.closed {
		return 0, fmt.Errorf("tunnel closed")
	}

	if len(buf) > t.mtu {
		return 0, fmt.Errorf("packet too large: %d > %d", len(buf), t.mtu)
	}

	n, err := syscall.Write(t.fd, buf)
	if err != nil {
		return 0, fmt.Errorf("write error: %v", err)
	}

	return n, nil
}

func (t *LinuxTunnel) Close() error {
	if t.closed {
		return nil
	}

	t.closed = false
	exec.Command("ip", "link", "set", "dev", t.name, "down").Run()
	return syscall.Close(t.fd)
}

func (t *LinuxTunnel) Name() string {
	return t.name
}