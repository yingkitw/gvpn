package routing

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
)

type RoutingManager struct {
	vpnNetwork string
	vpnGateway string
}

func New(vpnNetwork, vpnGateway string) *RoutingManager {
	return &RoutingManager{
		vpnNetwork: vpnNetwork,
		vpnGateway: vpnGateway,
	}
}

func (rm *RoutingManager) AddRoute(destNetwork string) error {
	switch runtime.GOOS {
	case "linux":
		return rm.addLinuxRoute(destNetwork)
	case "darwin":
		return rm.addDarwinRoute(destNetwork)
	case "windows":
		return rm.addWindowsRoute(destNetwork)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (rm *RoutingManager) DeleteRoute(destNetwork string) error {
	switch runtime.GOOS {
	case "linux":
		return rm.deleteLinuxRoute(destNetwork)
	case "darwin":
		return rm.deleteDarwinRoute(destNetwork)
	case "windows":
		return rm.deleteWindowsRoute(destNetwork)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (rm *RoutingManager) addLinuxRoute(destNetwork string) error {
	cmd := exec.Command("ip", "route", "add", destNetwork, "via", rm.vpnGateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add route: %v (output: %s)", err, string(output))
	}
	return nil
}

func (rm *RoutingManager) deleteLinuxRoute(destNetwork string) error {
	cmd := exec.Command("ip", "route", "del", destNetwork, "via", rm.vpnGateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete route: %v (output: %s)", err, string(output))
	}
	return nil
}

func (rm *RoutingManager) addDarwinRoute(destNetwork string) error {
	cmd := exec.Command("route", "add", "-net", destNetwork, rm.vpnGateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add route: %v (output: %s)", err, string(output))
	}
	return nil
}

func (rm *RoutingManager) deleteDarwinRoute(destNetwork string) error {
	cmd := exec.Command("route", "delete", "-net", destNetwork, rm.vpnGateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete route: %v (output: %s)", err, string(output))
	}
	return nil
}

func (rm *RoutingManager) addWindowsRoute(destNetwork string) error {
	cmd := exec.Command("route", "add", destNetwork, "mask", getNetmask(destNetwork), rm.vpnGateway)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add route: %v (output: %s)", err, string(output))
	}
	return nil
}

func (rm *RoutingManager) deleteWindowsRoute(destNetwork string) error {
	cmd := exec.Command("route", "delete", destNetwork)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete route: %v (output: %s)", err, string(output))
	}
	return nil
}

func getNetmask(network string) string {
	_, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return "255.255.255.255"
	}

	mask := ipnet.Mask
	return fmt.Sprintf("%d.%d.%d.%d", mask[0], mask[1], mask[2], mask[3])
}

func (rm *RoutingManager) AddDefaultRoutes(networks []string) error {
	for _, network := range networks {
		if err := rm.AddRoute(network); err != nil {
			return fmt.Errorf("failed to add route %s: %v", network, err)
		}
	}
	return nil
}

func (rm *RoutingManager) DeleteDefaultRoutes(networks []string) error {
	for _, network := range networks {
		if err := rm.DeleteRoute(network); err != nil {
			if !strings.Contains(err.Error(), "No such process") {
				return fmt.Errorf("failed to delete route %s: %v", network, err)
			}
		}
	}
	return nil
}

func (rm *RoutingManager) GetRoutingTable() ([]string, error) {
	switch runtime.GOOS {
	case "linux":
		return rm.getLinuxRoutingTable()
	case "darwin":
		return rm.getDarwinRoutingTable()
	case "windows":
		return rm.getWindowsRoutingTable()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (rm *RoutingManager) getLinuxRoutingTable() ([]string, error) {
	cmd := exec.Command("ip", "route", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get routing table: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

func (rm *RoutingManager) getDarwinRoutingTable() ([]string, error) {
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get routing table: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}

func (rm *RoutingManager) getWindowsRoutingTable() ([]string, error) {
	cmd := exec.Command("route", "print")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get routing table: %v", err)
	}

	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}