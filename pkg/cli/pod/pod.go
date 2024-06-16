package pod

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/duration"
	kubeclientset "k8s.io/client-go/kubernetes"

	"volcano.sh/apis/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/cli/util"
)

const (
	// Name pod name
	Name string = "Name"
	// Ready pod ready
	Ready string = "Ready"
	// Status pod status
	Status string = "Status"
	// Restart pod restart
	Restart string = "Restart"
	// Age pod age
	Age string = "Age"
)

type listFlags struct {
	util.CommonFlags
	// Namespace pod namespace
	Namespace string
	// JobName represents the pod created under this vcjob,
	// filtered by volcano.sh/job-name label
	// the default value is empty, which means
	// that all pods under vcjob will be obtained.
	JobName string
	// allNamespace represents getting all namespaces
	allNamespace bool
}

var listPodFlags = &listFlags{}

// InitListFlags init list command flags.
func InitListFlags(cmd *cobra.Command) {
	util.InitFlags(cmd, &listPodFlags.CommonFlags)

	cmd.Flags().StringVarP(&listPodFlags.JobName, "job", "j", "", "the name of job")
	cmd.Flags().StringVarP(&listPodFlags.Namespace, "namespace", "n", "default", "the namespace of job")
	cmd.Flags().BoolVarP(&listPodFlags.allNamespace, "all-namespaces", "", false, "list jobs in all namespaces")
}

// ListPods lists all pods details created by vcjob
func ListPods(ctx context.Context) error {
	config, err := util.BuildConfig(listPodFlags.Master, listPodFlags.Kubeconfig)
	if err != nil {
		return err
	}
	if listPodFlags.allNamespace {
		listPodFlags.Namespace = ""
	}

	labelSelector, err := createLabelSelector(listPodFlags.JobName)
	if err != nil {
		return err
	}

	client := kubeclientset.NewForConfigOrDie(config)
	pods, err := client.CoreV1().Pods(listPodFlags.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return err
	}

	// define the filter callback function based on different flags
	filterFunc := func(pod corev1.Pod) bool {
		// filter by namespace if specified
		if listPodFlags.Namespace != "" && listPodFlags.Namespace != pod.Namespace {
			return false
		}
		// filter by jobName if specified
		if listPodFlags.JobName != "" && listPodFlags.JobName != pod.Labels[v1alpha1.JobNameKey] {
			return false
		}
		return true
	}
	filteredPods := filterPods(pods, filterFunc)

	if len(filteredPods.Items) == 0 {
		fmt.Printf("No resources found\n")
		return nil
	}
	PrintPods(filteredPods, os.Stdout)

	return nil
}

func PrintPods(pods *corev1.PodList, writer io.Writer) {
	maxNameLen := 0
	maxReadyLen := 0
	maxStatusLen := 0
	maxRestartLen := 0
	maxAgeLen := 0

	var infoList []PodInfo
	for _, pod := range pods.Items {
		info, lens := printPod(&pod)
		infoList = append(infoList, info)

		if lens.Name > maxNameLen {
			maxNameLen = lens.Name
		}
		if lens.ReadyContainers > maxReadyLen {
			maxReadyLen = lens.ReadyContainers
		}
		if lens.Reason > maxStatusLen {
			maxStatusLen = lens.Reason
		}
		if lens.Restarts > maxRestartLen {
			maxRestartLen = lens.Restarts
		}
		if lens.CreationTimestamp > maxAgeLen {
			maxAgeLen = lens.CreationTimestamp
		}
	}
	columnSpacing := 8
	maxNameLen += columnSpacing
	maxReadyLen += columnSpacing
	maxStatusLen += columnSpacing
	maxRestartLen += columnSpacing
	maxAgeLen += columnSpacing
	formatStr := fmt.Sprintf("%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds\n", maxNameLen, maxReadyLen, maxStatusLen, maxRestartLen, maxAgeLen)
	_, err := fmt.Fprintf(writer, formatStr, Name, Ready, Status, Restart, Age)
	if err != nil {
		fmt.Printf("Failed to print Pod information: %s.\n", err)
		return
	}
	for _, info := range infoList {
		_, err := fmt.Fprintf(writer, formatStr, info.Name, info.ReadyContainers, info.Reason, info.Restarts, info.CreationTimestamp)
		if err != nil {
			fmt.Printf("Failed to print Pod information: %s.\n", err)
			return
		}
	}
}

// filterPods filters pods based on the provided filter callback function.
func filterPods(pods *corev1.PodList, filterFunc func(job corev1.Pod) bool) *corev1.PodList {
	filteredPods := &corev1.PodList{}
	for _, pod := range pods.Items {
		if filterFunc(pod) {
			filteredPods.Items = append(filteredPods.Items, pod)
		}
	}
	return filteredPods
}

// createLabelSelector creates a label selector based on the provided job name.
func createLabelSelector(jobName string) (labels.Selector, error) {
	var labelSelector labels.Selector

	if jobName == "" {
		inRequirement, err := labels.NewRequirement(v1alpha1.JobNameKey, selection.Exists, []string{})
		if err != nil {
			return nil, err
		}
		labelSelector = labels.NewSelector().Add(*inRequirement)
	} else {
		inRequirement, err := labels.NewRequirement(v1alpha1.JobNameKey, selection.In, []string{jobName})
		if err != nil {
			return nil, err
		}
		labelSelector = labels.NewSelector().Add(*inRequirement)
	}
	return labelSelector, nil
}

// translateTimestampSince translates a timestamp into a human-readable string using time.Since.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return duration.HumanDuration(time.Since(timestamp.Time))
}

// PodInfo holds information about a pod.
type PodInfo struct {
	Name              string
	ReadyContainers   string
	Reason            string
	Restarts          string
	CreationTimestamp string
}

// Lengths holds the maximum length of each column.
type Lengths struct {
	Name              int
	ReadyContainers   int
	Reason            int
	Restarts          int
	CreationTimestamp int
}

// printPod information in a tabular format.
func printPod(pod *corev1.Pod) (PodInfo, Lengths) {
	restarts := 0
	totalContainers := len(pod.Spec.Containers)
	readyContainers := 0
	lastRestartDate := metav1.NewTime(time.Time{})

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		if container.LastTerminationState.Terminated != nil {
			terminatedDate := container.LastTerminationState.Terminated.FinishedAt
			if lastRestartDate.Before(&terminatedDate) {
				lastRestartDate = terminatedDate
			}
		}
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		restarts = 0

		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.LastTerminationState.Terminated != nil {
				terminatedDate := container.LastTerminationState.Terminated.FinishedAt
				if lastRestartDate.Before(&terminatedDate) {
					lastRestartDate = terminatedDate
				}
			}
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				readyContainers++
			}
		}

	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == "NodeLost" {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	restartsStr := strconv.Itoa(restarts)
	if !lastRestartDate.IsZero() {
		restartsStr = fmt.Sprintf("%d (%s ago)", restarts, translateTimestampSince(lastRestartDate))
	}

	podInfo := PodInfo{
		Name:              pod.Name,
		ReadyContainers:   fmt.Sprintf("%d/%d", readyContainers, totalContainers),
		Reason:            reason,
		Restarts:          restartsStr,
		CreationTimestamp: translateTimestampSince(pod.CreationTimestamp),
	}

	lengths := Lengths{
		Name:              len(pod.Name),
		ReadyContainers:   len(fmt.Sprintf("%d/%d", readyContainers, totalContainers)),
		Reason:            len(reason),
		Restarts:          len(restartsStr),
		CreationTimestamp: len(translateTimestampSince(pod.CreationTimestamp)),
	}

	return podInfo, lengths
}
