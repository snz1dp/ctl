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

package kube

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	storageV1 "k8s.io/api/storage/v1"
	v1Errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
)

// GetPods 获取Pods
func GetPods(client kubernetes.Interface, namespace, selector string) ([]corev1.Pod, error) {
	list, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: selector,
	})
	return list.Items, err
}

// SelectorsForObject returns the pod label selector for a given object
//
// Modified version of https://github.com/kubernetes/kubernetes/blob/v1.14.1/pkg/kubectl/polymorphichelpers/helpers.go#L84
func SelectorsForObject(object runtime.Object) (selector labels.Selector, err error) {
	switch t := object.(type) {
	case *extensionsv1beta1.ReplicaSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1.ReplicaSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta2.ReplicaSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *corev1.ReplicationController:
		selector = labels.SelectorFromSet(t.Spec.Selector)
	case *appsv1.StatefulSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta1.StatefulSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta2.StatefulSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *extensionsv1beta1.DaemonSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1.DaemonSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta2.DaemonSet:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *extensionsv1beta1.Deployment:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1.Deployment:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta1.Deployment:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *appsv1beta2.Deployment:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *batchv1.Job:
		selector, err = metav1.LabelSelectorAsSelector(t.Spec.Selector)
	case *corev1.Service:
		if t.Spec.Selector == nil || len(t.Spec.Selector) == 0 {
			return nil, fmt.Errorf("invalid service '%s': Service is defined without a selector", t.Name)
		}
		selector = labels.SelectorFromSet(t.Spec.Selector)

	default:
		return nil, fmt.Errorf("selector for %T not implemented", object)
	}

	return selector, errors.Wrap(err, "invalid label selector")
}

// CreateNamespace creates a namespace using the given k8s interface.
func CreateNamespace(cs kubernetes.Interface, namespace string) error {
	if namespace == "" {
		// Setup default namespace
		namespace = "istio-system"
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: namespace,
	}}
	_, err := cs.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil && !v1Errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace %v: %v", namespace, err)
	}
	return nil
}

// CreateDockerRegistrySecret -
func CreateDockerRegistrySecret(cs kubernetes.Interface, ns string, registryKey string, registryURL string,
	username string, password string, labels map[string]string) error {
	const owner = "snz1dp"

	registrySecretVal := map[string]interface{}{
		"auths": map[string]interface{}{
			registryURL: map[string]string{
				"username": username,
				"password": password,
				"auth":     base64.StdEncoding.EncodeToString([]byte(username + ":" + password)),
			},
		},
	}

	jsonData, err := json.Marshal(registrySecretVal)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   registryKey,
			Labels: labels,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{".dockerconfigjson": jsonData},
	}
	_, err = cs.CoreV1().Secrets(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil && !v1Errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create secret: %v", err)
	}
	return nil
}

// CreateConfigMap -
func CreateConfigMap(cs kubernetes.Interface, ns string, mapKey string, vs map[string]string, labels map[string]string) error {
	secret := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   mapKey,
			Labels: labels,
		},
		Data: vs,
	}
	_, err := cs.CoreV1().ConfigMaps(ns).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil && !v1Errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create configmap: %v", err)
	}
	return nil
}

// ServerVersion -
func ServerVersion(cs kubernetes.Interface) (*version.Info, error) {
	return cs.Discovery().ServerVersion()
}

// GetStorageClassNames -
func GetStorageClassNames(cs kubernetes.Interface) (clazz map[string]bool, err error) {
	clazz = make(map[string]bool)
	var slt *storageV1.StorageClassList
	slt, err = cs.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return
	}

	for _, item := range slt.Items {
		clazz[item.GetName()] = true
		break
	}
	return
}

// DeleteValidatingWebhookConfiguration -
func DeleteValidatingWebhookConfiguration(cs kubernetes.Interface, name string) (err error) {
	return cs.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{})
}
