package rotator

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	typedappsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	typedpolicyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
)

type DrainOptions struct {
	// Continue even if there are pods not managed by a ReplicationController, ReplicaSet, Job, DaemonSet or StatefulSet.
	Force bool

	// Ignore DaemonSet-managed pods.
	IgnoreDaemonsets bool

	// Period of time in seconds given to each pod to terminate
	// gracefully.  If negative, the default value specified in the pod
	// will be used.
	GracePeriodSeconds int

	// The length of time to wait before giving up on deletion or
	// eviction.  Zero means infinite.
	Timeout time.Duration

	// Continue even if there are pods using emptyDir (local data that
	// will be deleted when the node is drained).
	DeleteLocalData bool

	// Namespace to filter pods on the node.
	Namespace string

	// Label selector to filter pods on the node.
	Selector labels.Selector

	// OnPodDeletedOrEvicted is called when a pod is evicted/deleted; for printing progress output
	OnPodDeletedOrEvicted func(pod *corev1.Pod, usingEviction bool)

	// SkipWaitForDeleteTimeoutSeconds ignores pods that have a
	// DeletionTimeStamp > N seconds. It's up to the user to decide when this
	// option is appropriate; examples include the Node is unready and the pods
	// won't drain otherwise
	SkipWaitForDeleteTimeoutSeconds int
}

type podDelete struct {
	pod    corev1.Pod
	status podDeleteStatus
}

type podDeleteStatus struct {
	delete  bool
	reason  string
	message string
}

type waitForDeleteParams struct {
	ctx                             context.Context
	pods                            []corev1.Pod
	interval                        time.Duration
	timeout                         time.Duration
	usingEviction                   bool
	getPodFn                        func(string, string) (*corev1.Pod, error)
	onDoneFn                        func(pod *corev1.Pod, usingEviction bool)
	globalTimeout                   time.Duration
	skipWaitForDeleteTimeoutSeconds int
}

// Takes a pod and returns a bool indicating whether or not to operate on the
// pod, an optional warning message, and an optional fatal error.
type podFilter func(corev1.Pod) (include bool, w *warning, f *fatal)
type warning struct {
	string
}
type fatal struct {
	string
}

const (
	EvictionKind        = "Eviction"
	EvictionSubresource = "pods/eviction"

	kDaemonsetFatal      = "DaemonSet-managed pods (use IgnoreDaemonsets to ignore)"
	kDaemonsetWarning    = "Ignoring DaemonSet-managed pods"
	kLocalStorageFatal   = "Pods with local storage (use DeleteLocalData to override)"
	kLocalStorageWarning = "Deleting pods with local storage"
	kUnmanagedFatal      = "Pods not managed by ReplicationController, ReplicaSet, Job, DaemonSet or StatefulSet (use Force to override)"
	kUnmanagedWarning    = "Deleting pods not managed by ReplicationController, ReplicaSet, Job, DaemonSet or StatefulSet"
)

func Drain(client kubernetes.Interface, nodes []*corev1.Node, options *DrainOptions, waitBetweenPodEvictions int) error {
	nodeInterface := client.CoreV1().Nodes()
	for _, node := range nodes {
		err := Cordon(nodeInterface, node)
		if err != nil {
			return err
		}
	}

	drainedNodes := sets.NewString()
	var fatal error

	for _, node := range nodes {
		err := DeleteOrEvictPods(client, node, options, waitBetweenPodEvictions)
		if err == nil {
			drainedNodes.Insert(node.Name)
			logger.Infof("Drained node %q", node.Name)
		} else {
			logger.WithError(err).Errorf("Unable to drain node %q", node.Name)
			remainingNodes := []string{}
			fatal = err
			for _, remainingNode := range nodes {
				if drainedNodes.Has(remainingNode.Name) {
					continue
				}
				remainingNodes = append(remainingNodes, remainingNode.Name)
			}

			if len(remainingNodes) > 0 {
				sort.Strings(remainingNodes)
				logger.Infof("There are pending nodes to be drained: %s", strings.Join(remainingNodes, ","))
			}
		}
	}

	return fatal
}

// DeleteOrEvictPods deletes or (where supported) evicts pods from the
// target node and waits until the deletion/eviction completes,
// Timeout elapses, or an error occurs.
func DeleteOrEvictPods(client kubernetes.Interface, node *corev1.Node, options *DrainOptions, waitBetweenPodEvictions int) error {
	pods, err := getPodsForDeletion(client, node, options)
	if err != nil {
		return err
	}
	err = deleteOrEvictPods(client, pods, options, waitBetweenPodEvictions)
	if err != nil {
		pendingPods, newErr := getPodsForDeletion(client, node, options)
		if newErr != nil {
			return newErr
		}
		pendingNames := make([]string, len(pendingPods))
		for i, pendingPod := range pendingPods {
			pendingNames[i] = pendingPod.Name
		}
		sort.Strings(pendingNames)
		logger.Errorf("Failed to evict pods from node %q (pending pods: %s): %v", node.Name, strings.Join(pendingNames, ","), err)
	}
	return err
}

func getPodController(pod corev1.Pod) *metav1.OwnerReference {
	return metav1.GetControllerOf(&pod)
}

func (o *DrainOptions) unreplicatedFilter(pod corev1.Pod) (bool, *warning, *fatal) {
	// any finished pod can be removed
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return true, nil, nil
	}

	controllerRef := getPodController(pod)
	if controllerRef != nil {
		return true, nil, nil
	}
	if o.Force {
		return true, &warning{kUnmanagedWarning}, nil
	}

	return false, nil, &fatal{kUnmanagedFatal}
}

type DaemonSetFilterOptions struct {
	client           typedappsv1.AppsV1Interface
	force            bool
	ignoreDaemonSets bool
}

func (o *DaemonSetFilterOptions) daemonSetFilter(pod corev1.Pod) (bool, *warning, *fatal) {
	ctx := context.TODO()

	// Note that we return false in cases where the pod is DaemonSet managed,
	// regardless of flags.  We never delete them, the only question is whether
	// their presence constitutes an error.
	//
	// The exception is for pods that are orphaned (the referencing
	// management resource - including DaemonSet - is not found).
	// Such pods will be deleted if Force is used.
	controllerRef := getPodController(pod)
	if controllerRef == nil || controllerRef.Kind != "DaemonSet" {
		return true, nil, nil
	}

	if _, err := o.client.DaemonSets(pod.Namespace).Get(ctx, controllerRef.Name, metav1.GetOptions{}); err != nil {
		// remove orphaned pods with a warning if Force is used
		if apierrors.IsNotFound(err) && o.force {
			return true, &warning{err.Error()}, nil
		}
		return false, nil, &fatal{err.Error()}
	}

	if !o.ignoreDaemonSets {
		return false, nil, &fatal{kDaemonsetFatal}
	}

	return false, &warning{kDaemonsetWarning}, nil
}

func mirrorPodFilter(pod corev1.Pod) (bool, *warning, *fatal) {
	if _, found := pod.ObjectMeta.Annotations[corev1.MirrorPodAnnotationKey]; found {
		return false, nil, nil
	}
	return true, nil, nil
}

func hasLocalStorage(pod corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			return true
		}
	}

	return false
}

func (o *DrainOptions) localStorageFilter(pod corev1.Pod) (bool, *warning, *fatal) {
	if !hasLocalStorage(pod) {
		return true, nil, nil
	}
	if !o.DeleteLocalData {
		return false, nil, &fatal{kLocalStorageFatal}
	}
	return true, &warning{kLocalStorageWarning}, nil
}

// Map of status message to a list of pod names having that status.
type podStatuses map[string][]string

func (ps podStatuses) message() string {
	msgs := []string{}

	for key, pods := range ps {
		msgs = append(msgs, fmt.Sprintf("%s: %s", key, strings.Join(pods, ", ")))
	}
	return strings.Join(msgs, "; ")
}

// getPodsForDeletion receives resource info for a node, and returns all the pods from the given node that we
// are planning on deleting. If there are any pods preventing us from deleting, we return that list in an error.
func getPodsForDeletion(client kubernetes.Interface, node *corev1.Node, options *DrainOptions) ([]corev1.Pod, error) {
	ctx := context.TODO()

	listOptions := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": node.Name}).String(),
	}
	if options.Selector != nil {
		listOptions.LabelSelector = options.Selector.String()
	}
	podList, err := client.CoreV1().Pods(options.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}

	ws := podStatuses{}
	fs := podStatuses{}

	daemonSetOptions := &DaemonSetFilterOptions{
		client:           client.AppsV1(),
		force:            options.Force,
		ignoreDaemonSets: options.IgnoreDaemonsets,
	}

	var pods []corev1.Pod
	for _, pod := range podList.Items {
		podOk := true
		for _, filt := range []podFilter{daemonSetOptions.daemonSetFilter, mirrorPodFilter, options.localStorageFilter, options.unreplicatedFilter} {
			filterOk, w, f := filt(pod)

			podOk = podOk && filterOk
			if w != nil {
				ws[w.string] = append(ws[w.string], pod.Name)
			}
			if f != nil {
				fs[f.string] = append(fs[f.string], pod.Name)
			}

			// short-circuit as soon as pod not ok
			// at that point, there is no reason to run pod
			// through any additional filters
			if !podOk {
				break
			}
		}
		if podOk {
			pods = append(pods, pod)
		}
	}

	if len(fs) > 0 {
		return []corev1.Pod{}, errors.New(fs.message())
	}
	if len(ws) > 0 {
		logger.Info(ws.message())
	}
	return pods, nil
}

func evictPod(client typedpolicyv1beta1.PolicyV1beta1Interface, pod corev1.Pod, policyGroupVersion string, gracePeriodSeconds int) error {
	ctx := context.TODO()

	deleteOptions := &metav1.DeleteOptions{}
	if gracePeriodSeconds >= 0 {
		gracePeriod := int64(gracePeriodSeconds)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}
	eviction := &policyv1beta1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyGroupVersion,
			Kind:       EvictionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: deleteOptions,
	}
	return client.Evictions(eviction.Namespace).Evict(ctx, eviction)
}

// deleteOrEvictPods deletes or evicts the pods on the api server
func deleteOrEvictPods(client kubernetes.Interface, pods []corev1.Pod, options *DrainOptions, waitBetweenPodEvictions int) error {
	ctx := context.TODO()

	if len(pods) == 0 {
		return nil
	}

	getPodFn := func(namespace, name string) (*corev1.Pod, error) {
		return client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	policyGroupVersion, err := SupportEviction(client)
	if err != nil {
		return err
	}

	if len(policyGroupVersion) > 0 {
		// Remember to change change the URL manipulation func when Evction's version change
		return evictPods(client.PolicyV1beta1(), pods, policyGroupVersion, options, getPodFn, waitBetweenPodEvictions)
	}
	return deletePods(client.CoreV1(), pods, options, getPodFn, waitBetweenPodEvictions)
}

func evictPods(client typedpolicyv1beta1.PolicyV1beta1Interface, pods []corev1.Pod, policyGroupVersion string, options *DrainOptions, getPodFn func(namespace, name string) (*corev1.Pod, error), waitBetweenPodEvictions int) error {
	returnCh := make(chan error, 1)
	// 0 timeout means infinite, we use MaxInt64 to represent it.
	var globalTimeout time.Duration
	if options.Timeout == 0 {
		globalTimeout = time.Duration(math.MaxInt64)
	} else {
		globalTimeout = options.Timeout * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()
	for _, pod := range pods {
		time.Sleep(time.Duration(waitBetweenPodEvictions) * time.Second)
		go func(pod corev1.Pod, returnCh chan error) {
			for {
				logger.Infof("Evicting pod %s/%s\n", pod.Namespace, pod.Name)
				select {
				case <-ctx.Done():
					// return here or we'll leak a goroutine.
					returnCh <- fmt.Errorf("Error when evicting pod %q: global timeout reached: %v", pod.Name, globalTimeout)
					return
				default:
				}
				err := evictPod(client, pod, policyGroupVersion, options.GracePeriodSeconds)
				if err == nil {
					break
				} else if apierrors.IsNotFound(err) {
					returnCh <- nil
					return
				} else if apierrors.IsTooManyRequests(err) {
					logger.Errorf("Error when evicting pod %q (will retry after 5s): %v\n", pod.Name, err)
					time.Sleep(5 * time.Second)
				} else {
					returnCh <- fmt.Errorf("Error when evicting pod %q: %v", pod.Name, err)
					return
				}
			}
			logger.Infof("Pod %s/%s evicted\n", pod.Namespace, pod.Name)
			params := waitForDeleteParams{
				ctx:                             ctx,
				pods:                            []corev1.Pod{pod},
				interval:                        1 * time.Second,
				timeout:                         time.Duration(math.MaxInt64),
				usingEviction:                   true,
				getPodFn:                        getPodFn,
				onDoneFn:                        options.OnPodDeletedOrEvicted,
				globalTimeout:                   globalTimeout,
				skipWaitForDeleteTimeoutSeconds: options.SkipWaitForDeleteTimeoutSeconds,
			}
			_, err := waitForDelete(params)
			if err == nil {
				returnCh <- nil
			} else {
				returnCh <- fmt.Errorf("Error when waiting for pod %q terminating: %v", pod.Name, err)
			}
		}(pod, returnCh)
	}

	doneCount := 0
	var errors []error

	numPods := len(pods)
	for doneCount < numPods {
		select {
		case err := <-returnCh:
			doneCount++
			if err != nil {
				errors = append(errors, err)
			}
		default:
		}
	}

	return utilerrors.NewAggregate(errors)
}

func deletePods(client typedcorev1.CoreV1Interface, pods []corev1.Pod, options *DrainOptions, getPodFn func(namespace, name string) (*corev1.Pod, error), waitBetweenPodEvictions int) error {
	// 0 timeout means infinite, we use MaxInt64 to represent it.
	var globalTimeout time.Duration
	if options.Timeout == 0 {
		globalTimeout = time.Duration(math.MaxInt64)
	} else {
		globalTimeout = options.Timeout * time.Second
	}
	for _, pod := range pods {
		time.Sleep(time.Duration(waitBetweenPodEvictions) * time.Second)
		err := DeletePod(client, pod)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	ctx := context.Background()
	params := waitForDeleteParams{
		ctx:                             ctx,
		pods:                            pods,
		interval:                        1 * time.Second,
		timeout:                         globalTimeout,
		usingEviction:                   false,
		getPodFn:                        getPodFn,
		onDoneFn:                        options.OnPodDeletedOrEvicted,
		globalTimeout:                   globalTimeout,
		skipWaitForDeleteTimeoutSeconds: options.SkipWaitForDeleteTimeoutSeconds,
	}
	_, err := waitForDelete(params)
	return err
}

// DeletePod will delete the given pod, or return an error if it couldn't
func DeletePod(client typedcorev1.CoreV1Interface, pod corev1.Pod) error {
	return client.Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
}

func waitForDelete(params waitForDeleteParams) ([]corev1.Pod, error) {
	pods := params.pods
	err := wait.PollImmediate(params.interval, params.timeout, func() (bool, error) {
		pendingPods := []corev1.Pod{}
		for i, pod := range pods {
			p, err := params.getPodFn(pod.Namespace, pod.Name)
			if apierrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID) {
				if params.onDoneFn != nil {
					params.onDoneFn(&pod, params.usingEviction)
				}
				continue
			} else if err != nil {
				return false, err
			} else {
				// if shouldSkipPod(*p, params.skipWaitForDeleteTimeoutSeconds) {
				// 	continue
				// }
				pendingPods = append(pendingPods, pods[i])
			}
		}
		pods = pendingPods
		if len(pendingPods) > 0 {
			select {
			case <-params.ctx.Done():
				return false, fmt.Errorf("Global timeout reached: %v", params.globalTimeout)
			default:
				return false, nil
			}
		}
		return true, nil
	})
	return pods, err
}

// SupportEviction uses Discovery API to find out if the server
// supports the eviction subresource.  If supported, it will return
// its groupVersion; otherwise it will return an empty string.
func SupportEviction(clientset kubernetes.Interface) (string, error) {
	discoveryClient := clientset.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}
	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == "policy" {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == EvictionSubresource && resource.Kind == EvictionKind {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}

// Cordon marks a node "Unschedulable".  This method is idempotent.
func Cordon(client typedcorev1.NodeInterface, node *corev1.Node) error {
	return cordonOrUncordon(client, node, true)
}

// Uncordon marks a node "Schedulable".  This method is idempotent.
func Uncordon(client typedcorev1.NodeInterface, node *corev1.Node) error {
	return cordonOrUncordon(client, node, false)
}

func cordonOrUncordon(client typedcorev1.NodeInterface, node *corev1.Node, desired bool) error {
	ctx := context.TODO()

	unsched := node.Spec.Unschedulable
	if unsched == desired {
		return nil
	}

	patch := []byte(fmt.Sprintf("{\"spec\":{\"unschedulable\":%t}}", desired))
	_, err := client.Patch(ctx, node.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err == nil {
		verbStr := "cordoned"
		if !desired {
			verbStr = "un" + verbStr
		}
		logger.Infof("%s node %q", verbStr, node.Name)
	}
	return err
}
