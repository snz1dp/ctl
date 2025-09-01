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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type waiter struct {
	c       kubernetes.Interface
	timeout time.Duration
	log     func(string, ...interface{})
}

func (w *waiter) waitComponentNotPods(ns string, dn string) error {
	return wait.Poll(2*time.Second, w.timeout, func() (bool, error) {
		lst, err := w.c.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=" + dn,
		})
		if err != nil {
			return false, err
		}

		w.log("app.kubernetes.io/name=%s lst count=%d", dn, len(lst.Items))

		if len(lst.Items) > 0 {
			return false, nil
		}

		lst, err = w.c.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
			LabelSelector: "app=" + dn,
		})

		if err != nil {
			return false, err
		}

		w.log("app=%s lst count=%d", dn, len(lst.Items))

		if len(lst.Items) > 0 {
			return false, nil
		}
		return true, nil
	})
}

func (w *waiter) waitDeploymentAvaible(ns string, dn string) error {
	return wait.Poll(2*time.Second, w.timeout, func() (bool, error) {
		var (
			err error
		)
		currentDeployment, err := w.c.AppsV1().Deployments(ns).Get(context.Background(), dn, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		// If paused deployment will never be ready
		if currentDeployment.Spec.Paused {
			return false, nil
		}
		// Find RS associated with deployment
		newReplicaSet, err := GetNewReplicaSet(currentDeployment, w.c.AppsV1())
		if err != nil || newReplicaSet == nil {
			return false, err
		}
		if !w.deploymentReady(newReplicaSet, currentDeployment) {
			return false, nil
		}
		return true, nil
	})
}

func (w *waiter) waitStatefulSetAvaible(ns string, dn string) error {
	return wait.Poll(2*time.Second, w.timeout, func() (bool, error) {
		var (
			err error
		)
		sts, err := w.c.AppsV1().StatefulSets(ns).Get(context.Background(), dn, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if !w.statefulSetReady(sts) {
			return false, nil
		}

		return true, nil
	})
}

func (w *waiter) deploymentReady(rs *appsv1.ReplicaSet, dep *appsv1.Deployment) bool {
	expectedReady := *dep.Spec.Replicas - MaxUnavailable(*dep)
	if !(rs.Status.ReadyReplicas >= expectedReady) {
		w.log("Deployment is not ready: %s/%s. %d out of %d expected pods are ready", dep.Namespace, dep.Name, rs.Status.ReadyReplicas, expectedReady)
		return false
	}
	return true
}

func (w *waiter) statefulSetReady(sts *appsv1.StatefulSet) bool {
	// If the update strategy is not a rolling update, there will be nothing to wait for
	if sts.Spec.UpdateStrategy.Type != appsv1.RollingUpdateStatefulSetStrategyType {
		return true
	}

	// Dereference all the pointers because StatefulSets like them
	var partition int
	// 1 is the default for replicas if not set
	var replicas = 1
	// For some reason, even if the update strategy is a rolling update, the
	// actual rollingUpdate field can be nil. If it is, we can safely assume
	// there is no partition value
	if sts.Spec.UpdateStrategy.RollingUpdate != nil && sts.Spec.UpdateStrategy.RollingUpdate.Partition != nil {
		partition = int(*sts.Spec.UpdateStrategy.RollingUpdate.Partition)
	}
	if sts.Spec.Replicas != nil {
		replicas = int(*sts.Spec.Replicas)
	}

	// Because an update strategy can use partitioning, we need to calculate the
	// number of updated replicas we should have. For example, if the replicas
	// is set to 3 and the partition is 2, we'd expect only one pod to be
	// updated
	expectedReplicas := replicas - partition

	// Make sure all the updated pods have been scheduled
	if int(sts.Status.UpdatedReplicas) != expectedReplicas {
		w.log("StatefulSet is not ready: %s/%s. %d out of %d expected pods have been scheduled", sts.Namespace, sts.Name, sts.Status.UpdatedReplicas, expectedReplicas)
		return false
	}

	if int(sts.Status.ReadyReplicas) != replicas {
		w.log("StatefulSet is not ready: %s/%s. %d out of %d expected pods are ready", sts.Namespace, sts.Name, sts.Status.ReadyReplicas, replicas)
		return false
	}
	return true
}

// LogFunc 日志
type LogFunc func(f string, v ...interface{})

// WaitDeploymentAvaible 等待部署
func WaitDeploymentAvaible(cs kubernetes.Interface, ns string, dn string, timeout time.Duration, log LogFunc) error {
	w := waiter{
		c:       cs,
		log:     log,
		timeout: timeout,
	}
	return w.waitDeploymentAvaible(ns, dn)
}

// WaitStatefulSetAvaible 等待部署
func WaitStatefulSetAvaible(cs kubernetes.Interface, ns string, dn string, timeout time.Duration, log LogFunc) error {
	w := waiter{
		c:       cs,
		log:     log,
		timeout: timeout,
	}
	return w.waitStatefulSetAvaible(ns, dn)
}

// WaitComponentNotPods 等待卸载
func WaitComponentNotPods(cs kubernetes.Interface, ns string, dn string, timeout time.Duration, log LogFunc) error {
	w := waiter{
		c:       cs,
		log:     log,
		timeout: timeout,
	}
	return w.waitComponentNotPods(ns, dn)
}
