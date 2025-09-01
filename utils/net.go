// *********************************************/
//                     _ooOoo_
//                    o8888888o
//                    88" . "88
//                    (| -_- |)
//                    O\  =  /O
//                 ____/`---'\____
//               .'  \\|     |//  `.
//              /  \\|||  :  |||//  \
//             /  _||||| -:- |||||-  \
//             |   | \\\  -  /// |   |
//             | \_|  ''\---/''  |   |
//             \  .-\__  `-`  ___/-. /
//           ___`. .'  /--.--\  `. . __
//        ."" '<  `.___\_<|>_/___.'  >'"".
//       | | :  `- \`.;`\ _ /`;.`/ - ` : | |
//       \  \ `-.   \_ __\ /__ _/   .-` /  /
//  ======`-.____`-.___\_____/___.-`____.-'======
//                     `=---='
// ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
//             佛祖保佑       永无BUG
//             心外无法       法外无心
//             三宝弟子       飞猪宏愿
// *********************************************/

package utils

import (
	"net"
)

// GetAvaibleInterfaces 获取可用的网卡接口
func GetAvaibleInterfaces() (ifs []net.Interface, err error) {
	var (
		ifaces []net.Interface
	)

	ifaces, err = net.Interfaces()
	if err != nil {
		return
	}

	for _, ift := range ifaces {
		if (ift.Flags&net.FlagUp) == 0 || (ift.Flags&net.FlagLoopback) != 0 {
			continue
		}
		ifs = append(ifs, ift)
	}
	return
}

// GetIpv4FromAddr 获取IPV4地址
func GetIpv4FromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil
	}
	return ip
}

// GetExternalIpv4 获取外部的IP
func GetExternalIpv4() (ip string) {
	ip = "127.0.0.1"
	var (
		ifs   []net.Interface
		addrs []net.Addr
		err   error
		ipv4  net.IP
	)

	if ifs, err = GetAvaibleInterfaces(); err != nil {
		return
	}

	for _, ifa := range ifs {
		if addrs, err = ifa.Addrs(); err != nil {
			continue
		}
		for _, addr := range addrs {
			ipv4 = GetIpv4FromAddr(addr)
			if ipv4 != nil {
				ip = ipv4.String()
			}
		}
	}

	return
}
