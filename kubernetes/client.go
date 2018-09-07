package kubernetes

import (
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

// CreateClient function is used to create a client for k8s and returns interface
func CreateClient(kubeconfig, context string) (*kubernetes.Clientset, error) {
	c, err := buildClientConf(kubeconfig, context)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(c)
}

func buildClientConf(kubeconfig, context string) (*rest.Config, error) {
	if kubeconfig != "" {
		info, err := os.Stat(kubeconfig)
		if err != nil || info.Size() == 0 {
			//If the specified file doesnot exist the it uses the default.
			kubeconfig = ""
		}
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubeconfig
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  context,
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
}

// CreateClientSet function is used to create a client for k8s and returns interface
func CreateClientSet(kubeconfig string) (kubernetes.Interface, error) {
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Errorf("build config from flags failed" + err.Error())
		return nil, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Errorf("new client from config failed" + err.Error())
		return nil, err
	}

	return client, nil
}

// UpdateConfigMap function is used to update the conf file.
func UpdateConfigMap(k kubernetes.Interface, conf, ns string) error {
	var (
		cConf []byte
		err   error
	)

	cObj := v1.ConfigMap{}

	cList, err := k.CoreV1().ConfigMaps(ns).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, cm := range cList.Items {
		if cm.Name == "mesher-configmap" {
			cObj.Name = "mesher-configmap"
			cObj.Namespace = ns
			cObj.Kind = "ConfigMap"
			cObj.APIVersion = "v1"
			cObj.Data = make(map[string]string)

			fInfo, err := ioutil.ReadDir(conf)
			if err != nil {
				return err
			}

			for _, f := range fInfo {
				cConf, err = ioutil.ReadFile(conf + f.Name())
				if err != nil {
					return err
				}
				cObj.Data[f.Name()] = string(cConf)
			}
		}
	}

	configMap, err := k.CoreV1().ConfigMaps(ns).Update(&cObj)
	if err != nil {
		return err
	}

	log.Infof("After Update configuration is:", configMap)

	return nil
}
