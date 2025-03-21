package router

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
)

type Router struct {
	// Define the fields of the Router struct here
}

var _ gin.HandlerFunc

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) Run(stop <-chan struct{}) {
	// Your application logic here

	// start router

	// start controller

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
