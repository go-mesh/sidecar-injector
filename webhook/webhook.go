package webhook

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"github.com/go-mesh/sidecar-injector/kubernetes"
	"github.com/howeyc/fsnotify"
	"k8s.io/api/admission/v1beta1"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/apis/core/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

const (
	webhookInjectKey          = "sidecar.mesher.io/inject"
	webhookStatusKey          = "sidecar.mesher.io/status"
	servicePortsAnnotationKey = "sidecar.mesher.io/servicePorts"
	registryTypeAnnotationKey = "sidecar.mesher.io/discoveryType"
	pilotConf                 = "pilot/"
	scConf                    = "sc/"
)

const (
	servicePorts = "SERVICE_PORTS"
)

//WebHookServer which has config contents
type WebHookServer struct {
	SidecarConfig *Config
	Server        *http.Server
	Watch         *fsnotify.Watcher
	Lock          sync.RWMutex
}

//WebHookParameters contains Server parameters
type WebHookParameters struct {
	Port                int
	KubeConfig          string
	CertFile            string
	KeyFile             string
	CAFile              string
	WebhookConfigName   string
	WebhookName         string
	SidecarConfigFile   string
	HealthCheckInterval time.Duration
	HealthCheckFile     string
}

//Config has container, volume and image information
type Config struct {
	Containers      []corev1.Container            `yaml:"containers"`
	Volumes         []corev1.Volume               `yaml:"volumes"`
	ImagePullSecret []corev1.LocalObjectReference `yaml:"imagePullSecrets"`
}

type operation struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value,omitempty"`
}

var (
	k     kube.Interface
	sList = make(map[string]int, 0)
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistration.AddToScheme(runtimeScheme)
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

func getKubeInterface(kubeConfig string) error {
	var err error

	if kubeConfig == "" {
		k, err = kubernetes.CreateClient(kubeConfig, "")
	} else {
		k, err = kubernetes.CreateClientSet(kubeConfig)
	}

	if err != nil {
		return err
	}

	return nil
}

//NewWebhook will load the configuration and create a server
func NewWebhook(p WebHookParameters) (*WebHookServer, error) {
	err := getKubeInterface(p.KubeConfig)
	if err != nil {
		return nil, err
	}

	sidecarConfig, err := loadConfig(p.SidecarConfigFile)
	if err != nil {
		log.Errorf("Filed to load configuration: %v", err)
		return nil, err
	}

	crt, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
	if err != nil {
		log.Errorf("Filed to load key pair: %v", err)
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("failed to create a watcher object: %v", err)
		return nil, err
	}

	for _, file := range []string{p.SidecarConfigFile, p.CertFile, p.KeyFile} {
		watchFile, _ := filepath.Split(file)
		if err := watcher.Watch(watchFile); err != nil {
			log.Errorf("failed to watch the files: %v", err)
			return nil, fmt.Errorf("could not watch %v: %v", file, err)
		}
	}

	wh := &WebHookServer{
		SidecarConfig: sidecarConfig,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", p.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{crt}},
		},
		Watch: watcher,
	}

	// define http server and server handler
	h := http.NewServeMux()
	h.HandleFunc("/webhookmutation", wh.webhookMutation)
	wh.Server.Handler = h

	return wh, nil
}

// (https://github.com/kubernetes/kubernetes/issues/57982)
func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume, secrets []corev1.LocalObjectReference) {
	defaulter.Default(&corev1.Pod{
		Spec: corev1.PodSpec{
			Containers:       containers,
			Volumes:          volumes,
			ImagePullSecrets: secrets,
		},
	})
}

func loadConfig(cfgFile string) (*Config, error) {
	data, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func requiredMutation(metaData *metav1.ObjectMeta) bool {
	annotations := metaData.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	status := annotations[webhookStatusKey]

	// determine whether to perform mutation based on annotation for the destination resource
	var mRequired bool
	if strings.ToLower(status) == "injected" {
		mRequired = false
	} else {
		switch strings.ToLower(annotations[webhookInjectKey]) {
		default:
			mRequired = false
		case "y", "yes":
			mRequired = true
		}
	}

	log.Infof("Mutation policy for %v/%v: status: %q required:%v", metaData.Namespace, metaData.Name, status, mRequired)
	return mRequired
}

func getAnnotations(metaData *metav1.ObjectMeta, key string) string {
	annotations := metaData.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	value, ok := annotations[key]
	if !ok {
		log.Errorf("key %s does not exist", key)
		return value
	}

	return value
}

func insertContainer(dest, add []corev1.Container, path string) (p []operation) {
	f := len(dest) == 0
	var val interface{}
	for _, add := range add {
		val = add
		path := path
		if f {
			f = false
			val = []corev1.Container{add}
		} else {
			path = path + "/-"
		}
		p = append(p, operation{
			Operation: "add",
			Path:      path,
			Value:     val,
		})
	}
	return p
}

func insertVolume(dest, add []corev1.Volume, path string) (p []operation) {
	f := len(dest) == 0
	var val interface{}
	for _, add := range add {
		val = add
		path := path
		if f {
			f = false
			val = []corev1.Volume{add}
		} else {
			path = path + "/-"
		}
		p = append(p, operation{
			Operation: "add",
			Path:      path,
			Value:     val,
		})
	}
	return p
}

func insertImagePullSecrets(dest, add []corev1.LocalObjectReference, path string) (p []operation) {
	f := len(dest) == 0
	var val interface{}
	for _, add := range add {
		val = add
		path := path
		if f {
			f = false
			val = []corev1.LocalObjectReference{add}
		} else {
			path = path + "/-"
		}
		p = append(p, operation{
			Operation: "add",
			Path:      path,
			Value:     val,
		})
	}
	return p
}

func annotationUpdate(dest map[string]string, add map[string]string) (p []operation) {
	for key, value := range add {
		if dest == nil || dest[key] == "" {
			dest = map[string]string{}
			p = append(p, operation{
				Operation: "add",
				Path:      "/metadata/annotations",
				Value: map[string]string{
					key: value,
				},
			})
		} else {
			p = append(p, operation{
				Operation: "replace",
				Path:      "/metadata/annotations/" + key,
				Value:     value,
			})
		}
	}
	return p
}

// create mutation patch for resoures
func createpatch(pod *corev1.Pod, sidecarConfig *Config, annotations map[string]string) ([]byte, error) {
	var p []operation

	p = append(p, insertContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
	p = append(p, insertVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
	p = append(p, insertImagePullSecrets(pod.Spec.ImagePullSecrets, sidecarConfig.ImagePullSecret, "/spec/imagePullSecrets")...)

	p = append(p, annotationUpdate(pod.Annotations, annotations)...)

	return json.Marshal(p)
}

// main mutation process
func (wh *WebHookServer) mutation(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Errorf("Could not unmarshal raw object: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, pod.Name, req.UID, req.Operation, req.UserInfo)

	// determine whether to perform mutation
	if !requiredMutation(&pod.ObjectMeta) {
		log.Infof("Skipping mutation for %s/%s due to policy check", pod.Namespace, pod.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	annotationValue := getAnnotations(&pod.ObjectMeta, registryTypeAnnotationKey)

	if annotationValue == "pilot" {
		err := kubernetes.UpdateConfigMap(k, pilotConf, req.Namespace)
		if err != nil {
			log.Errorf("Update configmap failed with error: ", err)
		}

		log.Infof("configmap updated successfully")
	}

	if annotationValue == "sc" {
		err := kubernetes.UpdateConfigMap(k, scConf, req.Namespace)
		if err != nil {
			log.Errorf("Update configmap failed with error: ", err)
		}

		log.Infof("configmap updated successfully")
	}

	// This sidecar container will be common to consumer and provider
	// So before proceeding first check the env "SERVICE_PORTS". If its present delete the env
	for k := range wh.SidecarConfig.Containers {
		for i := range wh.SidecarConfig.Containers[k].Env {
			if wh.SidecarConfig.Containers[k].Env[i].Name == servicePorts {
				wh.SidecarConfig.Containers[k].Env = append(wh.SidecarConfig.Containers[k].Env[0:i], wh.SidecarConfig.Containers[k].Env[i:]...)
			}
		}
	}

	// If annotation "sidecar.mesher.io/servicePorts" is provided then "SERVICE_PORTS" key will be added to sidecar container.
	sVal := getAnnotations(&pod.ObjectMeta, servicePortsAnnotationKey)
	if sVal != "" {
		for c := range wh.SidecarConfig.Containers {
			wh.SidecarConfig.Containers[c].Env = append(wh.SidecarConfig.Containers[c].Env, corev1.EnvVar{Name: servicePorts, Value: sVal})
		}
	}

	// Workaround: https://github.com/kubernetes/kubernetes/issues/57982
	applyDefaultsWorkaround(wh.SidecarConfig.Containers, wh.SidecarConfig.Volumes, wh.SidecarConfig.ImagePullSecret)
	annotations := map[string]string{webhookStatusKey: "injected"}
	patch, err := createpatch(&pod, wh.SidecarConfig, annotations)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("Response %v\n", string(patch))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patch,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

// Serve method for webhook server
func (wh *WebHookServer) webhookMutation(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		log.Errorf("empty request body")
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expect application/json", contentType)
		return
	}

	var aResponse *v1beta1.AdmissionResponse
	aRequest := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &aRequest); err != nil {
		log.Errorf("Can't decode body: %v", err)
		aResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		aResponse = wh.mutation(&aRequest)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if aResponse != nil {
		admissionReview.Response = aResponse
		if aRequest.Request != nil {
			admissionReview.Response.UID = aRequest.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
	}

	log.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
	}
}

//AddCABundle is used to add/update CABundle in MutatingWebhookConfiguration
func AddCABundle(p WebHookParameters) error {
	caPem, err := ioutil.ReadFile(p.CAFile)
	if err != nil {
		return err
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	wDir, _ := filepath.Split(p.CAFile)

	if err = w.Watch(wDir); err != nil {
		return fmt.Errorf("couldnot watch %v : %v", p.CAFile, err)
	}

	if err = updateCABundle(k, p.WebhookConfigName, p.WebhookName, caPem); err != nil {
		log.Errorf("failed to add ca bundle", err)
		return err
	}

	go func() {
		for {
			select {
			case <-w.Event:
				b, err := ioutil.ReadFile(p.CAFile)
				if err != nil {
					log.Errorf("failed to read CA bundle %v", err)
				} else {
					caPem = b
					if err = updateCABundle(k, p.WebhookConfigName, p.WebhookName, caPem); err != nil {
						log.Errorf("failed to add ca bundle", err)
					}
				}
			}
		}
	}()

	return nil
}

func updateCABundle(client kube.Interface, wConfigName, wName string, caBundle []byte) error {
	config, err := client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().
		Get(wConfigName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	originalConf, err := json.Marshal(config)
	if err != nil {
		return err
	}

	exist := false
	for i, w := range config.Webhooks {
		if w.Name == wName {
			config.Webhooks[i].ClientConfig.CABundle = caBundle
			exist = true
			break
		}
	}

	if !exist {
		return fmt.Errorf("webhook entry %s not found in %s", wName, wConfigName)
	}

	modifiedConf, err := json.Marshal(config)
	if err != nil {
		return err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(originalConf, modifiedConf,
		admissionregistration.MutatingWebhookConfiguration{})
	if err != nil {
		return err
	}

	if string(patch) != "{}" {
		_, err = client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().
			Patch(wConfigName, types.StrategicMergePatchType, patch)
	}

	return err
}

//Run will run the server
func (wh *WebHookServer) Run(stop <-chan struct{}, p WebHookParameters) {
	var healthChan <-chan time.Time

	if p.HealthCheckInterval != 0 && p.HealthCheckFile != "" {
		t := time.NewTicker(p.HealthCheckInterval)
		healthChan = t.C
		defer t.Stop()
	}

	go func() {
		if err := wh.Server.ListenAndServeTLS("", ""); err != nil {
			log.Errorf("Filed to listen and serve webhook server: %v", err)
		}
	}()

	defer wh.Server.Close()
	defer wh.Watch.Close()

	var timerChan <-chan time.Time

	for {
		select {
		case <-timerChan:
			sidecarConfig, err := loadConfig(p.SidecarConfigFile)
			if err != nil {
				log.Errorf("update error: %v", err)
				break
			}
			pair, err := tls.LoadX509KeyPair(p.CertFile, p.KeyFile)
			if err != nil {
				log.Errorf("reload cert error: %v", err)
				break
			}

			wh.Lock.Lock()
			wh.SidecarConfig = sidecarConfig
			wh.Server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{pair}}
			wh.Lock.Unlock()
		case event := <-wh.Watch.Event:
			if event.IsModify() || event.IsCreate() {
				timerChan = time.After(100 * time.Microsecond)
			}
		case err := <-wh.Watch.Error:
			log.Errorf("watcher error: %v", err)
		case <-healthChan:
			content := []byte(`ok`)
			if err := ioutil.WriteFile(p.HealthCheckFile, content, 0644); err != nil {
				log.Errorf("health check update of %q failed: %v", p.HealthCheckFile, err)
			}

		case <-stop:
			return
		}
	}
}
