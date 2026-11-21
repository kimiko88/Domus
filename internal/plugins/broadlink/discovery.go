package broadlink

import (
	"context"
	"fmt"
	"net"
	"sort"
	"syscall"
	"time"
)

func discover(ctx context.Context, target string) ([]DiscoveredDevice, error) {
	var localIPs []net.IP
	if target != "" {
		targetIP := net.ParseIP(target)
		if targetIP == nil || !targetIP.IsPrivate() {
			return nil, fmt.Errorf("broadlink: scan target must be a private LAN IP address")
		}
		routeConn, err := net.Dial("udp4", net.JoinHostPort(target, fmt.Sprint(udpPort)))
		if err != nil {
			return nil, fmt.Errorf("broadlink: determine LAN interface: %w", err)
		}
		local := routeConn.LocalAddr().(*net.UDPAddr).IP.To4()
		_ = routeConn.Close()
		if local == nil {
			return nil, fmt.Errorf("broadlink: could not select a local IPv4 interface")
		}
		localIPs = append(localIPs, local)
	} else {
		interfaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip4 := ip.To4(); ip4 != nil && ip4.IsPrivate() {
					localIPs = append(localIPs, ip4)
				}
			}
		}
	}
	if len(localIPs) == 0 {
		return nil, fmt.Errorf("broadlink: no private IPv4 network interface available")
	}

	found := make(map[string]DiscoveredDevice)
	for _, localIP := range localIPs {
		if ctx.Err() != nil {
			break
		}
		devices, err := discoverFromInterface(ctx, localIP, target)
		for _, device := range devices {
			found[device.MAC] = device
		}
		if err != nil && len(found) == 0 {
			continue
		}
	}
	result := make([]DiscoveredDevice, 0, len(found))
	for _, device := range found {
		result = append(result, device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].MAC < result[j].MAC })
	return result, nil
}

func discoverFromInterface(ctx context.Context, localIP net.IP, target string) ([]DiscoveredDevice, error) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: localIP})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	raw, err := conn.SyscallConn()
	if err == nil {
		_ = raw.Control(func(fd uintptr) {
			_ = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		})
	}
	local := conn.LocalAddr().(*net.UDPAddr)
	packet := discoveryPacket(localIP, local.Port, time.Now())
	destination := &net.UDPAddr{IP: net.IPv4bcast, Port: udpPort}
	if target != "" {
		destination.IP = net.ParseIP(target).To4()
	}
	if _, err := conn.WriteToUDP(packet, destination); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(1800 * time.Millisecond)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetReadDeadline(deadline)
	var devices []DiscoveredDevice
	buffer := make([]byte, 2048)
	for {
		n, from, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			if ctx.Err() != nil {
				break
			}
			return devices, err
		}
		if device, ok := parseDiscoveryResponse(buffer[:n], from); ok {
			devices = append(devices, device)
		}
	}
	return devices, nil
}
