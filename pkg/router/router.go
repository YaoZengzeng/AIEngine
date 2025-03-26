package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	aiv1alpha1 "AIEngine/api/v1alpha1"
	"AIEngine/internal/controller"
	"AIEngine/pkg/scheduler"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(aiv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type Router struct {
	// Define the fields of the Router struct here
	modelRouteController  *controller.ModelRouteReconciler
	modelServerController *controller.ModelServerReconciler

	scheduler scheduler.Scheduler
}

var _ gin.HandlerFunc

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) Run(stop <-chan struct{}) {
	// start controller
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		fmt.Printf("Unable to start manager")
		os.Exit(1)
	}

	mrc := (&controller.ModelRouteReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		ResourceToModels: make(map[string]string),
		Routes:           make(map[string][]*aiv1alpha1.Rule),
	})

	if mrc.SetupWithManager(mgr); err != nil {
		fmt.Printf("Unable to start Model Route Controller: %v", err)
		os.Exit(1)
	}
	r.modelRouteController = mrc

	msc := &controller.ModelServerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}

	if err := msc.SetupWithManager(mgr); err != nil {
		fmt.Printf("Unable to start Model Server Controller: %v", err)
		os.Exit(1)
	}
	r.modelServerController = msc

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Printf("Unable to start manager: %v", err)
	}

	r.scheduler = scheduler.NewScheduler()
}

type ModelRequest map[string]interface{}

func (r *Router) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		// implement gin request body reading here
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
		var modelRequest ModelRequest
		if err := json.Unmarshal(bodyBytes, &modelRequest); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, err)
			return
		}

		modelName, ok := modelRequest["model"].(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, "model not found")
		}

		fmt.Printf("Model name is %v\n", modelName)

		// find the ModelRoute, and get the endpoint model and address
		modelServerName, err := r.modelRouteController.Process(modelName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Sprintf("can't find corresponding model server: %v", err))
		}

		fmt.Printf("Model Server Name is %v\n", modelServerName)

		// according to modelRequest.Model, route to different model
		// call scheduler to select a model
		pods, err := r.modelServerController.Process(modelServerName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Sprintf("can't find target pods of model server: %v", modelServerName))
		}

		// call scheduler.ScheduleInference
		targetPod, err := r.scheduler.ScheduleInference(c.Request, pods)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Sprintf("can't schedule to target pod: %v", err))
		}

		req := c.Request

		original := req.URL.Host
		_, port, err := net.SplitHostPort(original)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, fmt.Sprintf("failed to split host and port from url host: %v", err))
		}

		// step 1: change request URL to real server URL.
		req.URL.Host = fmt.Sprintf("%s:%s", targetPod.Status.PodIP, port)

		// step 2: use http.Transport to do request to real server.
		transport := http.DefaultTransport
		resp, err := transport.RoundTrip(req)
		if err != nil {
			klog.Errorf("error: %v", err)
			c.String(500, "error")
			return
		}

		// step 3: return real server response to downstream.
		for k, vv := range resp.Header {
			for _, v := range vv {
				c.Header(k, v)
			}
		}
		defer resp.Body.Close()
		// Maybe we need to read the response to get the tokens for ratelimiting later
		bufio.NewReader(resp.Body).WriteTo(c.Writer)
	}
}
