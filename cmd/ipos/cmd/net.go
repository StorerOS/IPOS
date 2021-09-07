package cmd

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"syscall"

	"github.com/storeros/ipos/cmd/ipos/logger"
	xnet "github.com/storeros/ipos/pkg/net"
	"github.com/storeros/ipos/pkg/set"
)

func mustSplitHostPort(hostPort string) (host, port string) {
	xh, err := xnet.ParseHost(hostPort)
	if err != nil {
		logger.FatalIf(err, "Unable to split host port %s", hostPort)
	}
	return xh.Name, xh.Port.String()
}

func mustGetLocalIP4() (ipList set.StringSet) {
	ipList = set.NewStringSet()
	addrs, err := net.InterfaceAddrs()
	logger.FatalIf(err, "Unable to get IP addresses of this host")

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip.To4() != nil {
			ipList.Add(ip.String())
		}
	}

	return ipList
}

func mustGetLocalIP6() (ipList set.StringSet) {
	ipList = set.NewStringSet()
	addrs, err := net.InterfaceAddrs()
	logger.FatalIf(err, "Unable to get IP addresses of this host")

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip.To4() == nil {
			ipList.Add(ip.String())
		}
	}

	return ipList
}

func getHostIP(host string) (ipList set.StringSet, err error) {
	var ips []net.IP

	if ips, err = net.LookupIP(host); err != nil {
		return ipList, err
	}

	ipList = set.NewStringSet()
	for _, ip := range ips {
		ipList.Add(ip.String())
	}

	return ipList, err
}

type byLastOctetValue []net.IP

func (n byLastOctetValue) Len() int      { return len(n) }
func (n byLastOctetValue) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n byLastOctetValue) Less(i, j int) bool {
	if n[i].IsLoopback() {
		return false
	}
	if n[j].IsLoopback() {
		return true
	}
	return []byte(n[i].To4())[3] > []byte(n[j].To4())[3]
}

func sortIPs(ipList []string) []string {
	if len(ipList) == 1 {
		return ipList
	}

	var ipV4s []net.IP
	var nonIPs []string
	for _, ip := range ipList {
		nip := net.ParseIP(ip)
		if nip != nil {
			ipV4s = append(ipV4s, nip)
		} else {
			nonIPs = append(nonIPs, ip)
		}
	}

	sort.Sort(byLastOctetValue(ipV4s))

	var ips []string
	for _, ip := range ipV4s {
		ips = append(ips, ip.String())
	}

	return append(nonIPs, ips...)
}

func getAPIEndpoints() (apiEndpoints []string) {
	var ipList []string
	if globalIPOSHost == "" {
		ipList = sortIPs(mustGetLocalIP4().ToSlice())
		ipList = append(ipList, mustGetLocalIP6().ToSlice()...)
	} else {
		ipList = []string{globalIPOSHost}
	}

	for _, ip := range ipList {
		endpoint := fmt.Sprintf("%s://%s", getURLScheme(globalIsSSL), net.JoinHostPort(ip, globalIPOSPort))
		apiEndpoints = append(apiEndpoints, endpoint)
	}

	return apiEndpoints
}

func isHostIP(ipAddress string) bool {
	host, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		host = ipAddress
	}
	if i := strings.Index(host, "%"); i > -1 {
		host = host[:i]
	}
	return net.ParseIP(host) != nil
}

func checkPortAvailability(host, port string) (err error) {
	network := []string{"tcp", "tcp4", "tcp6"}
	for _, n := range network {
		l, err := net.Listen(n, net.JoinHostPort(host, port))
		if err == nil {
			if err = l.Close(); err != nil {
				return err
			}
		} else if errors.Is(err, syscall.EADDRINUSE) {
			return err
		}
	}

	return nil
}

func isLocalHost(host string, port string, localPort string) (bool, error) {
	hostIPs, err := getHostIP(host)
	if err != nil {
		return false, err
	}

	nonInterIPV4s := mustGetLocalIP4().Intersection(hostIPs)
	if nonInterIPV4s.IsEmpty() {
		hostIPs = hostIPs.ApplyFunc(func(ip string) string {
			if net.ParseIP(ip).IsLoopback() {
				return "127.0.0.1"
			}
			return ip
		})
		nonInterIPV4s = mustGetLocalIP4().Intersection(hostIPs)
	}
	nonInterIPV6s := mustGetLocalIP6().Intersection(hostIPs)

	isLocalv4 := !nonInterIPV4s.IsEmpty()
	isLocalv6 := !nonInterIPV6s.IsEmpty()
	if port != "" {
		return (isLocalv4 || isLocalv6) && (port == localPort), nil
	}
	return isLocalv4 || isLocalv6, nil
}

func CheckLocalServerAddr(serverAddr string) error {
	host, err := xnet.ParseHost(serverAddr)
	if err != nil {
		return err
	}

	if host.Name != "" && host.Name != net.IPv4zero.String() && host.Name != net.IPv6zero.String() {
		localHost, err := isLocalHost(host.Name, host.Port.String(), host.Port.String())
		if err != nil {
			return err
		}
		if !localHost {
			return errors.New("host in server address should be this server")
		}
	}

	return nil
}
