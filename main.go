package main

import (
	"context"
	"encoding/json"
	"errors"
	admissionv1 "k8s.io/api/admission/v1"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

type M map[string]interface{}

type Service struct {
	Metadata struct {
		Annotations *M `json:"annotations"`
	} `json:"metadata"`
	Spec struct {
		Type string `json:"type"`
	} `json:"spec"`
}

const (
	certFile = "/autoops-data/tls/tls.crt"
	keyFile  = "/autoops-data/tls/tls.key"
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

	cfgSubnet := strings.TrimSpace(os.Getenv("CFG_SUBNET"))
	if cfgSubnet == "" {
		err = errors.New("missing environment variable $LB_SUBNET")
		return
	}
	cfgMatchNamespace := strings.TrimSpace(os.Getenv("CFG_MATCH_NS"))
	if cfgMatchNamespace == "" {
		err = errors.New("missing environment variable $CFG_MATCH_NS")
		return
	}
	var matchNamespace *regexp.Regexp
	if matchNamespace, err = regexp.Compile(cfgMatchNamespace); err != nil {
		return
	}

	s := &http.Server{
		Addr: ":443",
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// decode request
			var review admissionv1.AdmissionReview
			if err := json.NewDecoder(req.Body).Decode(&review); err != nil {
				log.Println("Failed to decode a AdmissionReview:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			// log
			reviewPrettyJSON, _ := json.MarshalIndent(&review, "", "  ")
			log.Println(string(reviewPrettyJSON))

			// patches
			var buf []byte
			var svc Service

			if buf, err = review.Request.Object.MarshalJSON(); err != nil {
				log.Println("Failed to marshal object to json:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}
			if err = json.Unmarshal(buf, &svc); err != nil {
				log.Println("Failed to unmarshal object to service:", err.Error())
				http.Error(rw, err.Error(), http.StatusBadRequest)
				return
			}

			// build patches
			// build response
			var patchJSON []byte
			var patchType *admissionv1.PatchType

			if svc.Spec.Type == "LoadBalancer" && matchNamespace.MatchString(review.Request.Namespace) {
				var patches []M
				if svc.Metadata.Annotations == nil {
					patches = append(patches, M{
						"op":    "replace",
						"path":  "/metadata/annotations",
						"value": M{},
					})
				}
				patches = append(patches, M{
					"op":    "replace",
					"path":  "/metadata/annotations/service.kubernetes.io~1qcloud-loadbalancer-internal-subnetid",
					"value": cfgSubnet,
				})
				if patchJSON, err = json.Marshal(patches); err != nil {
					log.Println("Failed to marshal patches:", err.Error())
					http.Error(rw, err.Error(), http.StatusBadRequest)
					return
				}
				patchType = new(admissionv1.PatchType)
				*patchType = admissionv1.PatchTypeJSONPatch
			}

			review.Response = &admissionv1.AdmissionResponse{
				UID:       review.Request.UID,
				Allowed:   true,
				Patch:     patchJSON,
				PatchType: patchType,
			}
			review.Request = nil

			// send response
			reviewJSON, _ := json.Marshal(review)
			rw.Header().Set("Content-Type", "application/json")
			rw.Header().Set("Content-Length", strconv.Itoa(len(reviewJSON)))
			_, _ = rw.Write(reviewJSON)
		}),
	}

	// channels
	chErr := make(chan error, 1)
	chSig := make(chan os.Signal, 1)
	signal.Notify(chSig, syscall.SIGTERM, syscall.SIGINT)

	// start server
	go func() {
		log.Println("listening at :443")
		chErr <- s.ListenAndServeTLS(certFile, keyFile)
	}()

	// wait signal or failed start
	select {
	case err = <-chErr:
	case sig := <-chSig:
		log.Println("signal caught:", sig.String())
		_ = s.Shutdown(context.Background())
	}
}
