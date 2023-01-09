package utils

import (
	"bytes"
	"fmt"
	"net"
    "strconv"
	"strings"
)

// cidr contai
func IpInNetwork(ipstr string, cidr string) bool {
	ip := net.ParseIP(ipstr)
	if _, ipnet, err := net.ParseCIDR(cidr); err != nil {
		return false
	} else {
		return ipnet.Contains(ip)
	}
}

// allocate ip from cidr
func AllocIPFromCidr(cidr string, exist []string) (*string, error) {
	cidrIp, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	for ip := cidrIp.Mask(cidrNet.Mask); cidrNet.Contains(ip); inc(ip) {
		ipstr := ip.String()
		if !ListContains(exist, ipstr) {
			return &ipstr, nil
		}
	}
	return nil, fmt.Errorf("cidr is full")
}

// allocate ip from range
func AllocIPFromRange(start string, end string, exist []string) (*string, error) {
	startIp := net.ParseIP(start)
	endIp := net.ParseIP(end)

	for ip := startIp; bytes.Compare(ip, endIp) < 0; inc(ip) {
		ipstr := ip.String()
		if !ListContains(exist, ipstr) {
			return &ipstr, nil
		}
	}
	return nil, fmt.Errorf("ip range is full")
}

// inc ip
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// convert ipv4 to unit32
func Ipv4ToUint32(ipv4 string) (uint32, error) {
	var ipv4Uint32 uint32 = 0
	words := strings.Split(ipv4, ".")
	if len(words) != 4 {
		return 0, fmt.Errorf("wrong ipv4 format")
	}
	for i, word := range words {
		digit, err := strconv.ParseUint(word, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("wrong ipv4 format")
		}
		ipv4Uint32 += uint32(digit) << ((3 - i) * 8)
	}
	return ipv4Uint32, nil
}

// convert unit32 to ipv4
func Uint32ToIpv4(ipv4Uint32 uint32) string {
	ipv4 := ""
	var mask uint32 = 0xff000000
	for i := 0; i < 4; i++ {
		word := (ipv4Uint32 & mask) >> ((3 - i) * 8)
		ipv4 += fmt.Sprintf(".%d", word)
		mask >>= 8
	}

	return ipv4[1:]
}
