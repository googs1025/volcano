package pod

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
	"volcano.sh/volcano/pkg/cli/util"
)

func TestListPod(t *testing.T) {
	testCases := []struct {
		name           string
		Response       interface{}
		Namespace      string
		JobName        string
		ExpectedErr    error
		ExpectedOutput string
	}{
		{
			name: "Normal Case",
			Response: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "default",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			Namespace:   "default",
			JobName:     "",
			ExpectedErr: nil,
			ExpectedOutput: `Name          Ready      Status         Restart  Age       
my-pod        0/1        Running        0        0s`,
		},
		{
			name: "Normal Case with namespace filter",
			Response: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "default",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "kube-system",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			Namespace:   "default",
			JobName:     "",
			ExpectedErr: nil,
			ExpectedOutput: `Name          Ready      Status         Restart  Age       
my-pod        0/1        Running        0        0s`,
		},
		{
			name: "Normal Case with jobName filter",
			Response: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "default",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job1",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "default",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job2",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-pod",
							Namespace: "default",
							Labels: map[string]string{
								"volcano.sh/job-name": "my-job2",
							},
							CreationTimestamp: metav1.Now(),
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "my-container",
									Image: "nginx",
								},
							},
						},
						Status: corev1.PodStatus{
							Phase: corev1.PodRunning,
							Conditions: []corev1.PodCondition{
								{
									Type:   corev1.PodReady,
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			},
			Namespace:   "default",
			JobName:     "my-job2",
			ExpectedErr: nil,
			ExpectedOutput: `Name          Ready      Status         Restart  Age       
my-pod        0/1        Running        0        0s        
my-pod        0/1        Running        0        0s`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := util.CreateTestServer(testCase.Response)
			defer server.Close()
			// Set the server URL as the master flag
			listPodFlags.Master = server.URL
			listPodFlags.Namespace = testCase.Namespace
			listPodFlags.JobName = testCase.JobName
			listPodFlags.Namespace = testCase.Namespace
			r, oldStdout := util.RedirectStdout()
			defer r.Close()

			err := ListPods(context.TODO())
			gotOutput := util.CaptureOutput(r, oldStdout)

			if !reflect.DeepEqual(err, testCase.ExpectedErr) {
				t.Fatalf("test case: %s failed: got: %v, want: %v", testCase.name, err, testCase.ExpectedErr)
			}
			if gotOutput != testCase.ExpectedOutput {
				fmt.Println(gotOutput)
				t.Errorf("test case: %s failed: got: %s, want: %s", testCase.name, gotOutput, testCase.ExpectedOutput)
			}
		})
	}
}
