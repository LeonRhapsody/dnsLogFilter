package main

import (
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strings"
)

func GetIPAddress(interfaceName string) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	for _, iface := range interfaces {
		if iface.Name == interfaceName {
			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println(err)
				return ""
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if ok && !ipNet.IP.IsLoopback() {
					if ipNet.IP.To4() != nil {
						return formatIP(ipNet.IP.String())
					}
				}
			}
		}
	}

	return ""
}

// 删除IP中的点（.）并进行补位
func formatIP(ip string) string {

	// 将IP地址按小数点分隔
	ipSegments := strings.Split(ip, ".")

	// 补位，每个小段补足3位
	for i, segment := range ipSegments {
		ipSegments[i] = fmt.Sprintf("%03s", segment)
	}

	// 组装补位后的IP地址
	formattedIP := strings.Join(ipSegments, "")

	return formattedIP
}

func TestIP() {
	ipFormats := []string{
		"1.1.1.1",
		"1.1.1.1/24",
		"1.1.1.1-1.1.1.255",
	}

	for _, ipFormat := range ipFormats {
		ips := parseIPFormat(ipFormat)
		for _, ip := range ips {
			fmt.Println(ip)
		}
	}
}

func parseIPFormat(ipFormat string) []string {
	reCIDR := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)/(\d+)$`)
	reRange := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)-(\d+\.\d+\.\d+\.\d+)$`)

	// Check for CIDR notation
	if reCIDR.MatchString(ipFormat) {
		ip, ipNet, err := net.ParseCIDR(ipFormat)
		if err != nil {
			fmt.Println("Error parsing CIDR:", err)
			return nil
		}
		var ips []string
		for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
			ips = append(ips, ip.String())
		}
		return ips
	}

	// Check for IP range notation
	if reRange.MatchString(ipFormat) {
		parts := reRange.FindStringSubmatch(ipFormat)
		startIP := net.ParseIP(parts[1])
		endIP := net.ParseIP(parts[2])

		var ips []string
		for ip := startIP; ip.Equal(endIP) || bytesLessEqual(ip, endIP); inc(ip) {
			ips = append(ips, ip.String())
		}
		return ips
	}

	// Single IP address
	return []string{ipFormat}
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func bytesLessEqual(a, b net.IP) bool {
	return bytesLess(a, b) || a.Equal(b)
}

func bytesLess(a, b net.IP) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	return false
}

// 将IPv6地址转换为大整数
func ipToBigInt(ip string) *big.Int {
	// 解析IPv6地址
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		fmt.Println("Invalid IPv6 address")
		return nil
	}

	// 将IPv6地址转换为字节切片
	ipBytes := ipAddr.To16()

	// 将字节切片转换为大整数
	ipInt := new(big.Int).SetBytes(ipBytes)
	return ipInt
}

func (t *TaskInfo) v6WithInRange(ip string) bool {

	// IPv6地址段
	ipRangeStart := net.ParseIP("")
	ipRangeEnd := net.ParseIP("")

	// 要判断的IPv6地址
	ipToCheck := net.ParseIP(ip)

	// 判断地址是否在段内
	if isIPv6InRange(ipToCheck, ipRangeStart, ipRangeEnd) {
		return true

	} else {

	}

	return false

}

// 判断IPv6地址是否在给定的IPv6地址段内
func isIPv6InRange(ip, start, end net.IP) bool {
	// 检查地址是否在范围内
	if bytesLessEqualIP(start, ip) && bytesLessEqualIP(ip, end) {
		return true
	}
	return false
}

// 字节比较函数
func bytesLessEqualIP(a, b net.IP) bool {
	return bytesLessIP(a, b) || a.Equal(b)
}

func bytesLessIP(a, b net.IP) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] {
			return true
		} else if a[i] > b[i] {
			return false
		}
	}
	return false
}
