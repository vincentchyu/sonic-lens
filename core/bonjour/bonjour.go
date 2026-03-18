package bonjour

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/grandcat/zeroconf"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// Start 在局域网内广播 SonicLens 服务
func Start(ctx context.Context, cfg config.BonjourConfig, portStr string) func() {
	if !cfg.Enabled {
		log.Info(ctx, "Bonjour 广播已禁用")
		return func() {}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		log.Warn(ctx, "Bonjour 广播端口非法", zap.String("port", portStr), zap.Error(err))
		return func() {}
	}

	serviceType := cfg.ServiceType
	if serviceType == "" {
		serviceType = "_soniclens._tcp"
	}
	if serviceType[len(serviceType)-1] != '.' {
		serviceType = serviceType + "."
	}

	domain := cfg.Domain
	if domain == "" {
		domain = "local."
	}
	domain = strings.TrimSuffix(domain, ".")

	instanceName := cfg.Name
	if instanceName == "" {
		host, _ := os.Hostname()
		if host == "" {
			instanceName = "SonicLens"
		} else {
			instanceName = fmt.Sprintf("SonicLens (%s)", host)
		}
	}

	txt := []string{
		"api=soniclens",
		"version=1",
		"path=/",
		"ws=/ws",
		"health=/health",
	}

	hostName := buildBonjourHostName(domain)
	ips := listServiceIPs()

	server, err := zeroconf.RegisterProxy(instanceName, serviceType, domain, port, hostName, ips, txt, nil)
	if err != nil {
		log.Error(ctx, "Bonjour 广播启动失败", zap.Error(err))
		return func() {}
	}

	log.Info(
		ctx, "Bonjour 广播已启动",
		zap.String("name", instanceName),
		zap.String("service", serviceType),
		zap.String("domain", domain),
		zap.String("host", hostName),
		zap.Strings("ips", ips),
		zap.Int("port", port),
	)

	return func() {
		server.Shutdown()
		log.Info(ctx, "Bonjour 广播已停止")
	}
}

// buildBonjourHostName 规范化主机名，避免出现 .local.local. 这类重复后缀。
func buildBonjourHostName(domain string) string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "soniclens"
	}

	host = strings.TrimSuffix(host, ".")
	host = strings.TrimSuffix(host, "."+domain)

	return fmt.Sprintf("%s.%s.", host, domain)
}

// listServiceIPs 返回本机可用于 Bonjour 广播的非回环单播地址。
func listServiceIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	ips := make([]string, 0, 4)
	seen := make(map[string]struct{})

	for _, iface := range ifaces {
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

			if ip == nil || ip.IsLoopback() || !ip.IsGlobalUnicast() {
				continue
			}

			ip = ip.To16()
			if ip == nil {
				continue
			}

			ipStr := ip.String()
			if _, ok := seen[ipStr]; ok {
				continue
			}

			seen[ipStr] = struct{}{}
			ips = append(ips, ipStr)
		}
	}

	return ips
}
