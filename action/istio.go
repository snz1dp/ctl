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

package action

import (
	"istio.io/istio/istioctl/pkg/kubernetes"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const (
	installedSpecCRPrefix = "installed-state"
	defaultIstioNamespace = "istio-system"
	currentIstioVersion   = "1.6.5"
)

func newKubeExecClient(restConfig *rest.Config) (kubeClient *kubernetes.Client, err error) {
	config := *restConfig
	config.APIPath = "/api"
	config.GroupVersion = &v1.SchemeGroupVersion
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	restClient, err := rest.RESTClientFor(&config)
	kubeClient = &kubernetes.Client{&config, restClient, currentIstioVersion}
	return
}

// func getRemoteInfo(restConfig *rest.Config, istioNamespace string, istioVersion string) (*istioVer.MeshInfo, error) {
// 	config := *restConfig
// 	config.APIPath = "/api"
// 	config.GroupVersion = &v1.SchemeGroupVersion
// 	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

// 	restClient, err := rest.RESTClientFor(&config)
// 	if err != nil {
// 		return nil, err
// 	}
// 	kubeClient := &kubernetes.Client{&config, restClient, istioVersion}
// 	return kubeClient.GetIstioVersions(istioNamespace)
// }
