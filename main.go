package main

import (
	"context"
	"encoding/json"
	"github.com/k8s-autoops/autoops"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"net/http"
	"os"
)

const (
	AnnotationKeySubnet = "autoops.enforce-qcloud-internal-lb/subnet"
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	} else {
		log.Println("exited")
	}
}

func main() {
	var err error
	defer exit(&err)

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var client *kubernetes.Clientset
	if client, err = autoops.InClusterClient(); err != nil {
		return
	}

	s := &http.Server{
		Addr: ":443",
		Handler: autoops.NewMutatingAdmissionHTTPHandler(
			func(ctx context.Context, request *admissionv1.AdmissionRequest, patches *[]map[string]interface{}) (err error) {
				var buf []byte
				if buf, err = request.Object.MarshalJSON(); err != nil {
					return
				}
				var svc corev1.Service
				if err = json.Unmarshal(buf, &svc); err != nil {
					return
				}
				// 如果不是 LoadBalancer 则忽略
				if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
					return
				}
				// 获取命名空间并检查特定注解
				var ns *corev1.Namespace
				if ns, err = client.CoreV1().Namespaces().Get(ctx, request.Namespace, metav1.GetOptions{}); err != nil {
					return
				}
				if ns.Annotations == nil {
					return
				}
				if ns.Annotations[AnnotationKeySubnet] == "" {
					return
				}
				// 增加注解
				if svc.Annotations == nil {
					*patches = append(*patches, map[string]interface{}{
						"op":    "replace",
						"path":  "/metadata/annotations",
						"value": map[string]interface{}{},
					})
				}
				*patches = append(*patches, map[string]interface{}{
					"op":    "replace",
					"path":  "/metadata/annotations/service.kubernetes.io~1qcloud-loadbalancer-internal-subnetid",
					"value": ns.Annotations[AnnotationKeySubnet],
				})
				return
			},
		),
	}

	if err = autoops.RunAdmissionServer(s); err != nil {
		return
	}
}
