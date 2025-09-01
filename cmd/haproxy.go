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

package cmd

import (
	"github.com/spf13/cobra"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func newStartHaProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStartHaproxy(setting)
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start haproxy service on local",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&a.GenerateHaproxyCfg, "generate-config", true, "generate haproxy.cfg file")
	fs.BoolVarP(&a.ForcePullImage, "force-pull", "f", false, "force pull haproxy image")
	fs.BoolVar(&a.LoadImageLocal, "load-image", false, "load local image to docker")

	return cmd
}

func newReStartHaProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewStopHaproxy(setting)
	a := action.NewStartHaproxy(setting)
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart haproxy service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.Run()
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&a.GenerateHaproxyCfg, "generate-config", true, "generate haproxy.cfg file")
	fs.BoolVarP(&a.ForcePullImage, "force-pull", "f", false, "force pull haproxy image")

	return cmd
}

func newStopHaProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewStopHaproxy(setting)
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "stop haproxy service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newListHaProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewListHaproxy(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list haproxy proxy services",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}
	return cmd
}

func newHaProxyProxyAddCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewAddProxyService(setting)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "proxy service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.ServiceName == "" || a.Port <= 0 || len(a.Backends) == 0 {
				return cmd.Usage()
			}
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&a.ServiceName, "name", "n", "", "proxy service name")
	fs.IntVarP(&a.Port, "port", "p", 0, "proxy service port")
	fs.BoolVar(&a.AccessProxy, "access-proxy", false, "use access-proxy listen")
	fs.BoolVar(&a.SendProxy, "send-proxy", false, "use send-proxy to backend")
	fs.StringVarP(&a.Mode, "mode", "m", "http", "proxy service mode")
	fs.StringVar(&a.Balance, "balance", "source", "proxy service balance mode")
	fs.IntVar(&a.Inter, "inter", 1500, "inter of proxy service backend")
	fs.IntVar(&a.Rise, "rise", 3, "rise of proxy service backend")
	fs.IntVar(&a.Fall, "fall", 3, "fall of proxy service backend")
	fs.StringArrayVarP(&a.Backends, "backend", "b", []string{}, "backends ip and port list, ex: 192.168.1.2:80")
	return cmd
}

func newHaProxyProxyRemoveCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewRemoveProxyService(setting)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "proxy service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(a.ServiceName) == 0 && len(args) == 0 {
				return cmd.Usage()
			}
			a.ServiceName = append(a.ServiceName, args...)
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringArrayVarP(&a.ServiceName, "name", "n", []string{}, "proxy service name")
	return cmd
}

func newHaProxyProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "handle proxy service",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newHaProxyProxyAddCmd(setting),
		newHaProxyProxyRemoveCmd(setting),
	)
	return cmd
}

func newHaProxyBackendAddCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewAddBackend(setting)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "backend",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.Service == "" || a.BackendName == "" || a.IP == "" || a.Port <= 0 {
				return cmd.Usage()
			}
			return a.Run()
		},
	}

	fs := cmd.Flags()

	fs.StringVarP(&a.Service, "service", "s", "", "proxy service name")
	fs.StringVarP(&a.BackendName, "name", "n", "", "backend name of proxy service")
	fs.StringVarP(&a.IP, "ip", "i", "", "backend ip address")
	fs.IntVarP(&a.Port, "port", "p", 0, "backend port")
	fs.BoolVar(&a.SendProxy, "send-proxy", false, "use send-proxy to backend")

	return cmd
}

func newHaProxyBackendRemoveCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewRemoveBackend(setting)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "backend",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if a.Service == "" || a.Port <= 0 {
				return cmd.Usage()
			}
			return a.Run()
		},
	}
	fs := cmd.Flags()
	fs.StringVarP(&a.Service, "service", "s", "", "proxy service name")
	fs.StringVarP(&a.IP, "ip", "i", "", "backend ip address")
	fs.IntVarP(&a.Port, "port", "p", 0, "backend port")

	return cmd
}

func newHaProxyBackendCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backend",
		Short: "handle backends",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newHaProxyBackendAddCmd(setting),
		newHaProxyBackendRemoveCmd(setting),
	)
	return cmd
}

func newHaProxyLogCmd(setting *action.GlobalSetting) *cobra.Command {
	a := action.NewLogHaproxy(setting)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "fetch logs of haproxy",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.Run()
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&a.Since, "since", "", "show logs since timestamp  (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	fs.StringVar(&a.Tail, "tail ", "all", "number of lines to show from the end of the logs")
	fs.BoolVarP(&a.Follow, "follow", "f", false, "Follow log output")
	fs.BoolVar(&a.Details, "details", false, "show extra details provided to logs")
	fs.BoolVarP(&a.Timestamps, "timestamps", "t", false, "show extra details provided to logs")
	fs.StringVar(&a.Until, "until", "", "Show logs before a timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")

	return cmd
}

func newHaProxyCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "haproxy",
		Short: "haproxy command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newStartHaProxyCmd(setting),
		newStopHaProxyCmd(setting),
		newHaProxyLogCmd(setting),
		newReStartHaProxyCmd(setting),
		newListHaProxyCmd(setting),
		newHaProxyProxyCmd(setting),
		newHaProxyBackendCmd(setting),
	)
	return cmd
}
