package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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
	"AIEngine/internal/limiter/redis"
	"AIEngine/internal/picker"
	"AIEngine/internal/router"
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

	limiter, err := redis.NewRateLimiter()
	if err != nil {
		fmt.Printf("unable to construct rate limiter")
		os.Exit(1)
	}

	picker, err := picker.NewEndpointPicker()
	if err != nil {
		fmt.Printf("unable to construct endpoint picker")
		os.Exit(1)
	}

	router, err := router.NewModelRouter()
	if err != nil {
		fmt.Printf("unable to constrcut model router")
		os.Exit(1)
	}

	mrc := (&controller.ModelRouteReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),

		ResourceToModels: make(map[string]string),

		ModelRouter:    router,
		EndpointPicker: picker,
		RateLimiter:    limiter,
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

		// according to modelRequest.Model, route to different model
		// call scheduler to select a model

		// call scheduler.ScheduleInference
		var endpointAddr string

		req := c.Request
		// step 1: change request URL to real server URL.
		req.URL.Host = endpointAddr

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
