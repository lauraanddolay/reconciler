package data

import (
	"testing"

	"github.com/avast/retry-go"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_Gatherer_GetAllPods(t *testing.T) {
	firstPod := fixPodWith("application", "kyma", "istio/proxyv2:1.10.1", "Running")
	secondPod := fixPodWith("istio", "custom", "istio/proxyv2:1.10.2", "Terminating")
	retryOpts := getTestingRetryOptions()

	t.Run("should not get any pod from the cluster when there are no pods running", func(t *testing.T) {
		// given
		kubeClient := fake.NewSimpleClientset()
		gatherer := DefaultGatherer{}

		// when
		pods, err := gatherer.GetAllPods(kubeClient, retryOpts)

		// then
		require.NoError(t, err)
		require.Empty(t, pods)
	})

	t.Run("should get all pods from the cluster when there are pods running", func(t *testing.T) {
		// given
		kubeClient := fake.NewSimpleClientset(firstPod, secondPod)
		gatherer := DefaultGatherer{}

		// when
		pods, err := gatherer.GetAllPods(kubeClient, retryOpts)

		// then
		require.NoError(t, err)
		require.Len(t, pods.Items, 2)
	})
}

func Test_Gatherer_GetPodsWithDifferentImage(t *testing.T) {
	image := ExpectedImage{
		Prefix:  "istio/proxyv2",
		Version: "1.10.1",
	}
	podWithExpectedImage := fixPodWith("application", "kyma", "istio/proxyv2:1.10.1", "Running")
	podWithExpectedImageTerminating := fixPodWith("istio", "custom", "istio/proxyv2:1.10.2", "Terminating")
	podWithExpectedImagePending := fixPodWith("istio", "custom", "istio/proxyv2:1.10.2", "Pending")
	podWithDifferentImageSuffix := fixPodWith("istio", "custom", "istio/proxyv2:1.10.2", "Running")
	podWithDifferentImageSuffixTerminating := fixPodWith("application", "kyma", "istio/proxyv2:1.10.2", "Terminating")
	podWithDifferentImageSuffixPending := fixPodWith("application", "kyma", "istio/proxyv2:1.10.2", "Pending")
	podWithDifferentImagePrefix := fixPodWith("application", "kyma", "istio/weirdimage:1.10.2", "Running")

	t.Run("should not get any pods from an empty list", func(t *testing.T) {
		// given
		var pods v1.PodList
		gatherer := DefaultGatherer{}

		// when
		podsWithDifferentImage := gatherer.GetPodsWithDifferentImage(pods, image)

		// then
		require.Empty(t, podsWithDifferentImage.Items)
	})

	t.Run("should get one pod from the list", func(t *testing.T) {
		// given
		var pods v1.PodList
		var expected v1.PodList
		pods.Items = []v1.Pod{
			*podWithExpectedImage,
			*podWithExpectedImageTerminating,
			*podWithExpectedImagePending,
			*podWithDifferentImageSuffix,
			*podWithDifferentImageSuffixTerminating,
			*podWithDifferentImageSuffixPending,
			*podWithDifferentImagePrefix,
		}
		expected.Items = []v1.Pod{
			*podWithDifferentImageSuffix,
		}
		gatherer := DefaultGatherer{}

		// when
		podsWithDifferentImage := gatherer.GetPodsWithDifferentImage(pods, image)

		// then
		require.Equal(t, podsWithDifferentImage.Items, expected.Items)
		require.NotEmpty(t, podsWithDifferentImage.Items)
		require.Len(t, podsWithDifferentImage.Items, 1)
	})
}

func fixPodWith(name, namespace, image, phase string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ReplicaSet"},
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: v1.PodStatus{
			Phase: v1.PodPhase(phase),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  name + "-container",
					Image: "wrongimage:6.9",
				},
				{
					Name:  name + "-containertwo",
					Image: image,
				},
			},
		},
	}
}

func getTestingRetryOptions() []retry.Option {
	return []retry.Option{
		retry.Delay(0),
		retry.Attempts(1),
		retry.DelayType(retry.FixedDelay),
	}
}
