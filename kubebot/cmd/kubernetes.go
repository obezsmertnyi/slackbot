package cmd

//
import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// clientset is a global variable that holds the Kubernetes clientset, allowing
// interactions with Kubernetes API server.
var clientset *kubernetes.Clientset

// initKubernetesClient initializes the Kubernetes clientset used for interacting
// with the Kubernetes cluster.
func initKubernetesClient() error {
	var config *rest.Config
	var err error

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		log.Println("Using KUBECONFIG for configuration")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Println("Trying to build configuration from environment variables")
		config, err = buildConfigFromEnvVars()
		if err != nil {
			log.Println("Falling back to in-cluster configuration")
			config, err = rest.InClusterConfig()
		}
	}

	if err != nil {
		log.Printf("Failed to configure Kubernetes client: %v", err)
		return err
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Failed to create Kubernetes clientset: %v", err)
		return err
	}

	return nil
}

func ensureConnected() error {
	_, err := clientset.Discovery().ServerVersion()
	if err != nil {
		log.Println("Connection lost. Attempting to reconnect...")
		return initKubernetesClient()
	}
	return nil
}

func buildConfigFromEnvVars() (*rest.Config, error) {
	server := os.Getenv("KUBE_SERVER")
	token := os.Getenv("KUBE_TOKEN")
	caData := os.Getenv("KUBE_CA")

	if server == "" || token == "" || caData == "" {
		return nil, fmt.Errorf("KUBE_SERVER, KUBE_CA, and KUBE_TOKEN environment variables must be set")
	}

	caDecoded, err := base64.StdEncoding.DecodeString(caData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode KUBE_CA: %w", err)
	}

	config := &rest.Config{
		Host:        server,
		BearerToken: token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caDecoded,
		},
	}

	return config, nil
}

// checkPodStatusAfterPromotion monitors pods in a namespace to confirm successful deployment of a target version.
func checkPodStatusAfterPromotion(namespace, targetVersion string, command string, client *slack.Client, channelID, userID string) {
	// create watcher for pods in a namespace
	watcher, err := clientset.CoreV1().Pods(namespace).Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to watch pods in namespace `%s`: %v", namespace, err)
		sendErrorMessage(client, channelID, userID, command, fmt.Sprintf("Failed to watch pods in namespace `%s`", namespace))
		return
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			log.Println("Unexpected type")
			continue
		}

		version, err := extractPodVersion(pod)
		if err != nil {
			log.Printf("Failed to extract version from pod `%s` in namespace `%s`", pod.Name, namespace)
			continue // Skip if version extraction fails.
		}

		// Check if the pod with the updated version is launched and running
		if event.Type == watch.Added || event.Type == watch.Modified {
			if pod.Status.Phase == corev1.PodRunning && version == targetVersion {
				sendSuccessMessage(client, channelID, userID, command, fmt.Sprintf("Pod `%s` with version `%s` in namespace `%s` is successfully running.", pod.Name, version, namespace))
				return
			} else if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
				sendErrorMessage(client, channelID, userID, command, fmt.Sprintf("Pod `%s` with version `%s` in namespace `%s` has failed to start.", pod.Name, version, namespace))
				// Continue monitoring; failure of one pod does not imply failure of promotion.
			}
		}
	}
}

// getPodsInfoWithRetries attempts to retrieve information about pods in a specified namespace
// with a specified number of retries and a delay between retries.
func getPodsInfoWithRetries(namespace string, maxRetries int, retryDelay time.Duration) ([]string, []string, []string, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		// Attempt to get information about the pods
		podNames, versions, labelSelectors, err := getPodsInfo(namespace)
		if err == nil {
			return podNames, versions, labelSelectors, nil // Успішно отримали інформацію, повертаємо результат
		}

		// Log the error and wait before the next attempt
		log.Printf("Error getting pods info in namespace '%s', attempt %d/%d: %v", namespace, i+1, maxRetries, err)
		lastErr = err
		time.Sleep(retryDelay)
	}

	// Return the last error after exhausting all attempts
	return nil, nil, nil, fmt.Errorf("failed to get pods info in namespace '%s' after %d attempts: %v", namespace, maxRetries, lastErr)
}

// getPodsInfo retrieves information about pods in a specified namespace, including
// their names, versions extracted from container images, and label selectors.
func getPodsInfo(namespace string) ([]string, []string, []string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	if len(pods.Items) == 0 {
		return nil, nil, nil, fmt.Errorf("no pods found in namespace %s", namespace)
	}

	var podNames []string
	var versions []string
	var labelSelectors []string

	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)

		var version string
		for _, container := range pod.Spec.Containers {
			parts := strings.Split(container.Image, ":")
			if len(parts) > 1 {
				version = parts[1]
				break
			}
		}
		versions = append(versions, version)

		labels := pod.Labels
		labelSelector := labels["app.kubernetes.io/name"]
		labelSelectors = append(labelSelectors, labelSelector)
	}

	return podNames, versions, labelSelectors, nil
}

// GetPodStatus retrieves the status of the specified pod in the given namespace.
func GetPodStatus(podName, namespace string) (string, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get pod details: %w", err)
	}

	// Перевіряємо статус кожного контейнера в поді
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason != "" {
			// Return pod phase if containers are not waiting.(ex, CrashLoopBackOff)
			return containerStatus.State.Waiting.Reason, nil
		}
	}

	// Return pod phase if containers are not waiting.
	return string(pod.Status.Phase), nil
}

// extractPodVersion extracts the version from the pod's containers.
func extractPodVersion(pod *corev1.Pod) (string, error) {
	for _, container := range pod.Spec.Containers {
		// Assuming the version is included in the image name (e.g., "grafana/loki:2.6.1")
		parts := strings.Split(container.Image, ":")
		if len(parts) == 2 {
			return parts[1], nil // Return version if found in image name.
		}
	}
	return "", fmt.Errorf("version not found")
}
