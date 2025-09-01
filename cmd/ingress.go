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
	flag "github.com/spf13/pflag"
	"snz1.cn/snz1dp/snz1dpctl/action"
)

func addIngressPublicParam(fs *flag.FlagSet, s *action.BaseIngressAction) {
	fs.StringVar(&s.GatewayAdminURL, "admin-url", "", "url of snz1dp ingress admin")
	fs.StringVar(&s.RSAPrivateKey, "private-key", "", "private key of snz1dp ingress admin")
	fs.StringVar(&s.KeyID, "admin-keyid", "", "keyid of snz1dp ingress admin")
}

func newIngressCreateJwtConsumerCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewCreateJwtConsumer(setting)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create consumer and jwt auth",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if (s.ConsumerID == "" && len(args) == 0) || (s.ConsumerID != "" && len(args) > 0) {
				return cmd.Usage()
			} else if len(args) == 1 {
				s.ConsumerID = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.ConsumerID, "name", "n", "", "name of consumer")
	fs.StringVarP(&s.KeyID, "keyid", "k", "", "keyid of jwt auth, default is name of consumer")
	fs.StringVarP(&s.RSAPublicKey, "pubkey", "p", "", "pem file of rsa public or pem context")
	fs.StringArrayVarP(&s.Group, "group", "g", []string{}, "group of consumer")
	fs.BoolVarP(&s.Override, "override", "y", false, "override old consumer or jwt auth")
	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of route")

	return cmd
}

func newIngressDeleteConsumerCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressDeleteConsumer(setting)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete consumer or jwt auth",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.ConsumerID = append(s.ConsumerID, args...)
			if len(s.ConsumerID) == 0 || (len(s.ConsumerID) > 1 && len(s.KeyID) > 0) {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringArrayVarP(&s.ConsumerID, "appid", "n", []string{}, "name or id of consumer")
	fs.StringArrayVarP(&s.KeyID, "keyid", "k", []string{}, "name or id of jwt auth keyid")
	fs.BoolVarP(&s.Force, "yes", "y", false, "confirm to delete consumer or jwt auth")

	return cmd
}

func newIngressListConsumerCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressListAppConsumer(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list consumer and jwt auth keyid",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.ConsumerID = append(s.ConsumerID, args...)
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.OutputFormat, "output-format", "o", "table", "output format of consumer")
	fs.StringArrayVarP(&s.ConsumerID, "name", "n", []string{}, "name of consumer")
	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of consumer")
	fs.BoolVarP(&s.MatchAllTags, "match-all-tags", "a", false, "match all tags")

	return cmd
}

func newIngressAddServiceCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressAddService(setting)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "create backend service defined",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.ServiceName == "" && len(args) == 0 || s.ServiceName != "" && len(args) != 0 {
				return cmd.Usage()
			} else if s.ServiceName == "" {
				s.ServiceName = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.ServicePath, "path", "P", "", "path of service")
	fs.IntVar(&s.ReadTimeout, "read-timeout", 60000, "read timeout of service")
	fs.IntVar(&s.WriteTimeout, "write-timeout", 60000, "write timeout of service")
	fs.IntVar(&s.Retries, "retry-times", 5, "retry times of service")

	fs.StringVarP(&s.ServiceName, "name", "n", "", "service name")
	fs.StringVarP(&s.BackendHost, "host", "H", "", "backend host name")
	fs.IntVarP(&s.BackendPort, "port", "p", 80, "backend host port")
	fs.BoolVarP(&s.Override, "override", "y", false, "override old service")

	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of service")

	return cmd
}

func newIngressListServicesCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressListService(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list defined backend services",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.ServiceName = append(s.ServiceName, args...)
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.OutputFormat, "output-format", "o", "table", "output format of service")
	fs.StringArrayVarP(&s.ServiceName, "name", "n", []string{}, "name of service")
	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of service")
	fs.BoolVarP(&s.MatchAllTags, "match-all-tags", "a", false, "match all tags")
	return cmd
}

func newIngressDeleteServicesCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressDeleteService(setting)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete backend service",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.ServiceNames = append(s.ServiceNames, args...)
			if len(s.ServiceNames) == 0 {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.BoolVarP(&s.Force, "yes", "y", false, "confirm to delete service")
	fs.StringArrayVarP(&s.ServiceNames, "name", "n", []string{}, "name or id of service")
	return cmd
}

func newIngressServiceCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "snz1dp ingress service manager command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressAddServiceCmd(setting),
		newIngressListServicesCmd(setting),
		newIngressDeleteServicesCmd(setting),
	)
	return cmd
}

func newIngressRouteCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route",
		Short: "snz1dp ingress route manager command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressAddRouteCmd(setting),
		newIngressListRoutesCmd(setting),
		newIngressDeleteRoutesCmd(setting),
	)
	return cmd
}

func newIngressListRoutesCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressListRoute(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list defined routes",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.RouteName = append(s.RouteName, args...)
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.OutputFormat, "output-format", "o", "table", "output format of service")
	fs.StringArrayVarP(&s.RouteName, "name", "n", []string{}, "name of route")
	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of route")
	fs.BoolVarP(&s.MatchAllTags, "match-all-tags", "a", false, "match all tags")
	return cmd
}

func newIngressAddRouteCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressAddRoute(setting)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create service route defined",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.RouteName == "" && len(args) == 0 || s.RouteName != "" && len(args) != 0 {
				return cmd.Usage()
			} else if s.RouteName == "" {
				s.RouteName = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.RouteName, "name", "n", "", "name of service route")
	fs.StringArrayVar(&s.Domain, "host", []string{}, "domain name of route")
	fs.StringArrayVarP(&s.Path, "path", "P", []string{}, "context path of route")
	fs.StringArrayVar(&s.Protocol, "protocol", []string{}, "protocols of route")
	fs.BoolVar(&s.PreserveHost, "preserve-host", true, "preserve host name for route")
	fs.BoolVar(&s.StripPath, "strip-path", false, "strip route path request backend")

	fs.StringVar(&s.ServicePath, "service-path", "", "path of service")
	fs.IntVar(&s.ReadTimeout, "read-timeout", 60000, "read timeout of service")
	fs.IntVar(&s.WriteTimeout, "write-timeout", 60000, "write timeout of service")
	fs.IntVar(&s.Retries, "retry-times", 5, "retry times of service")

	fs.StringVarP(&s.ServiceName, "service-name", "s", "", "service name")
	fs.StringVarP(&s.BackendHost, "backend-host", "H", "", "backend host name of service")
	fs.IntVarP(&s.BackendPort, "backend-port", "p", 80, "backend host port of service")

	fs.StringVar(&s.AuthType, "auth-mode", "", "auth mode of route")
	fs.StringArrayVar(&s.AllowGroup, "allow-group", []string{}, "allow role groups to access route")
	fs.StringArrayVar(&s.DenyGroup, "deny-group", []string{}, "deny role groups to access route")

	fs.BoolVarP(&s.Override, "override", "y", false, "override old route")
	fs.BoolVar(&s.AllowAnonymous, "allow-anonymous", false, "sso allow anonymous access")

	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of route")

	return cmd
}

func newIngressDeleteRoutesCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressDeleteRoute(setting)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete service route",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.RouteNames = append(s.RouteNames, args...)
			if len(s.RouteNames) == 0 {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.BoolVarP(&s.Force, "yes", "y", false, "confirm to delete route")
	fs.StringArrayVarP(&s.RouteNames, "name", "n", []string{}, "name or id of route")
	return cmd
}

func newIngressAppuserCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consumer",
		Short: "snz1dp ingress consumer manager command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressCreateJwtConsumerCmd(setting),
		newIngressListConsumerCmd(setting),
		newIngressDeleteConsumerCmd(setting),
	)
	return cmd
}

func newIngressUpstreamCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upstream",
		Short: "snz1dp ingress upstream manager command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressListUpstreamCmd(setting),
		newIngressAddUpstreamCmd(setting),
		newIngressDeleteUpstreamCmd(setting),
		newIngressUpstreamTargetCmd(setting),
	)
	return cmd
}

func newIngressApplyCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressApply(setting)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "apply ingress config file",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.ServiceName == "" {
				return cmd.Usage()
			}
			if s.ComponentFile == "" && len(args) == 0 || s.ComponentFile != "" && len(args) != 0 {
				return cmd.Usage()
			} else if s.ComponentFile == "" {
				s.ComponentFile = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	addIngressPublicParam(fs, &s.BaseIngressAction)
	fs.StringVarP(&s.ComponentFile, "run-config-file", "f", "", "run config file")
	fs.StringVarP(&s.ServiceName, "service-name", "s", "", "service name of ingress")

	return cmd
}

func newIngressLoginCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressLoginConfig(setting)
	cmd := &cobra.Command{
		Use:   "config",
		Short: "setup ingress default login config",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.DefaultURL == "" || s.LoginURL == "" || s.LogoutURL == "" {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()
	addIngressPublicParam(fs, &s.BaseIngressAction)
	fs.StringVarP(&s.DefaultURL, "default-url", "d", "", "website default url")
	fs.StringVarP(&s.LoginURL, "login-url", "l", "", "default login url")
	fs.StringVarP(&s.LogoutURL, "logout-url", "o", "", "default logout url")

	return cmd
}

func newIngressUpstreamTargetCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "upstream target manager command tool",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressUpstreamAddTargetCmd(setting),
		newIngressUpstreamRemoveTargetCmd(setting),
	)
	return cmd
}

func newIngressUpstreamAddTargetCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressAddUpstreamTarget(setting)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "add upstream target",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.Target = append(s.Target, args...)
			if s.UpstreamName == "" || len(s.Target) == 0 {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.UpstreamName, "upstream", "u", "", "name of upstream")
	fs.IntVarP(&s.Weight, "weight", "w", 100, "weight of target")
	fs.StringArrayVarP(&s.Target, "target", "t", []string{}, "target ip and port(format: ip:port)")
	return cmd
}

func newIngressUpstreamRemoveTargetCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressRemoveUpstreamTarget(setting)
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "remove upstream target",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.Target = append(s.Target, args...)
			if s.UpstreamName == "" || len(s.Target) == 0 {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.UpstreamName, "upstream", "u", "", "name of upstream")
	fs.StringArrayVarP(&s.Target, "target", "t", []string{}, "target ip and port(format: ip:port)")
	fs.BoolVarP(&s.Force, "really", "y", false, "really to remove upstream target")
	return cmd
}

func newIngressListUpstreamCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressListUpstream(setting)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list defined upstreams",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.UpstreamName = append(s.UpstreamName, args...)
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.OutputFormat, "output-format", "o", "table", "output format of service")
	fs.StringArrayVarP(&s.UpstreamName, "name", "n", []string{}, "name of route")
	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of route")
	fs.BoolVarP(&s.MatchAllTags, "match-all-tags", "a", false, "match all tags")
	return cmd
}

func newIngressAddUpstreamCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressAddUpstream(setting)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "create upstream defined",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			if s.UpstreamName == "" && len(args) == 0 || s.UpstreamName != "" && len(args) != 0 {
				return cmd.Usage()
			} else if s.UpstreamName == "" {
				s.UpstreamName = args[0]
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.StringVarP(&s.UpstreamName, "name", "n", "", "upsteam name")
	fs.StringVarP(&s.HealthPath, "health-path", "p", "/health", "health path of upstream")
	fs.StringVarP(&s.Algorithm, "algorithm", "A", "round-robin", "algorithm of upstream(consistent-hashing, least-connections, round-robin)")
	fs.IntVarP(&s.Timeout, "health-timeout", "T", 3, "health timeout seconds of upstream")
	fs.IntVar(&s.HealthInterval, "health-interval", 300, "health interval of upstream")
	fs.IntVar(&s.Successes, "healthy-successes", 1, "healthy successes of upstream")
	fs.IntVar(&s.UnhealthyTimeouts, "unhealthy-timeouts", 3, "unhealthy timeouts of upstream")
	fs.IntVar(&s.HTTPFailures, "unhealthy-http-failures", 3, "unhealthy http failures of upstream")
	fs.IntVar(&s.TCPFailures, "unhealthy-tcp-failures", 3, "unhealthy tcp failures of upstream")
	fs.BoolVar(&s.HTTPSVerifyCertificate, "https-verify-cert", false, "https verify certificate upstream")
	fs.IntVarP(&s.Slots, "slots", "s", 10000, "slots of upstream")
	fs.BoolVarP(&s.Override, "override", "y", false, "override old upstream")

	fs.StringArrayVarP(&s.Tags, "tag", "t", []string{}, "tags of upstream")

	return cmd
}

func newIngressDeleteUpstreamCmd(setting *action.GlobalSetting) *cobra.Command {
	s := action.NewIngressDeleteUpstream(setting)
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete upstream",
		Long:  globalBanner,
		RunE: func(cmd *cobra.Command, args []string) error {
			s.UpstreamName = append(s.UpstreamName, args...)
			if len(s.UpstreamName) == 0 {
				return cmd.Usage()
			}
			return s.Run()
		},
	}

	fs := cmd.Flags()

	addIngressPublicParam(fs, &s.BaseIngressAction)

	fs.BoolVarP(&s.Force, "yes", "y", false, "confirm to delete upstream")
	fs.StringArrayVarP(&s.UpstreamName, "name", "n", []string{}, "name or id of upstream")
	return cmd
}

func newIngressCmd(setting *action.GlobalSetting) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingress",
		Short: "snz1dp ingress config tool command",
		Long:  globalBanner,
	}
	cmd.AddCommand(
		newIngressServiceCmd(setting),
		newIngressAppuserCmd(setting),
		newIngressRouteCmd(setting),
		newIngressUpstreamCmd(setting),
		newIngressApplyCmd(setting),
		newIngressLoginCmd(setting),
	)
	return cmd
}
