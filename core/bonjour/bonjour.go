package bonjour

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	"go.uber.org/zap"

	"github.com/vincentchyu/sonic-lens/config"
	"github.com/vincentchyu/sonic-lens/core/log"
)

// Start 在局域网内广播 SonicLens 服务，并自动巡检 IP 变动。
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
	// zeroconf 库不要求也不建议 serviceType 以点结尾，多余的点会导致 "bad rdata"
	serviceType = strings.TrimSuffix(serviceType, ".")

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

	var (
		mu            sync.Mutex
		currentServer *zeroconf.Server
		currentIPs    []string
	)

	// updateRegistration 执行具体的注册/重新注册逻辑
	updateRegistration := func() {
		mu.Lock()
		defer mu.Unlock()

		newIPs := listIPs()
		// 如果 IP 没变且服务已在运行，则跳过
		if reflect.DeepEqual(currentIPs, newIPs) && currentServer != nil {
			return
		}

		// 如果已有服务，先关闭
		if currentServer != nil {
			currentServer.Shutdown()
			currentServer = nil
			log.Info(
				ctx, "检测到 IP 变动，正在重新注册 Bonjour 广播", zap.Strings("old_ips", currentIPs),
				zap.Strings("new_ips", newIPs),
			)
		}

		if len(newIPs) == 0 {
			log.Warn(ctx, "未发现可用局域网 IP，无法注册 Bonjour 广播")
			currentIPs = nil
			return
		}

		// 使用 Register 而非 RegisterProxy，代码中明确指定 "local." 避免某些环境下的歧义
		server, err := zeroconf.Register(instanceName, serviceType, "local.", port, txt, nil)
		if err != nil {
			log.Error(ctx, "Bonjour 广播注册失败", zap.Error(err))
			return
		}

		currentServer = server
		currentIPs = newIPs

		log.Info(
			ctx, "Bonjour 广播已激活",
			zap.String("name", instanceName),
			zap.String("service", serviceType),
			// zap.String("domain", domain),
			zap.Strings("ips", currentIPs),
			zap.Int("port", port),
		)
	}

	// 初始注册
	updateRegistration()

	// 启动定时巡检协程（每分钟检查一次 IP）
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				updateRegistration()
			}
		}
	}()

	return func() {
		mu.Lock()
		if currentServer != nil {
			currentServer.Shutdown()
			log.Info(ctx, "Bonjour 广播已停止")
		}
		mu.Unlock()
	}
}

// listIPs 仅用于比对网络状态是否发生变化
func listIPs() []string {
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
