package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/pipelineruns"
	"google.golang.org/grpc/codes"
)

func exposeHTTPServer(ctx context.Context, r Resolver, port string) {
	serv := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        resolverHandler(r),
		ReadTimeout:    20 * time.Second, // git clones of a repo can take a while so these timeouts are long.
		WriteTimeout:   20 * time.Second, // caching todo.
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(serv.ListenAndServe())
}

func resolverHandler(r Resolver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println("received request", req.URL.String())
		b, err := ioutil.ReadAll(req.Body)
		if err != nil {
			writeError(w, codes.Unknown, fmt.Errorf("error reading request body: %w", err))
			return
		}
		resreq := pipelineruns.ResolutionInterceptorRequest{}
		err = json.Unmarshal(b, &resreq)
		if err != nil {
			writeError(w, codes.Unknown, fmt.Errorf("error parsing json: %v", err))
			return
		}

		params := resreq.Params
		log.Printf("params: %#v", resreq.Params)

		err = r.ValidateParams(params)
		if err != nil {
			writeError(w, codes.Unknown, fmt.Errorf("error validating: %w", err))
			return
		}

		data, annotations, err := r.Resolve(params)
		if err != nil {
			writeError(w, codes.Unknown, fmt.Errorf("error resolving: %w", err))
			return
		}

		for key, val := range annotations {
			w.Header().Set(http.CanonicalHeaderKey("x-"+key), val)
		}

		resp := pipelineruns.ResolutionInterceptorResponse{}
		resp.Resolved = data
		respBytes, jsonErr := json.Marshal(resp)
		if jsonErr == nil {
			w.Write(respBytes)
		} else {
			log.Printf("error serializing to json: %v", jsonErr)
			w.Write([]byte(`{"error":"unknown"}`))
		}
	})
}

func writeError(w http.ResponseWriter, code codes.Code, err error) {
	w.Header().Set(http.CanonicalHeaderKey("content-type"), "application/json")
	resp := pipelineruns.ResolutionInterceptorResponse{
		Status: pipelineruns.Status{
			Code:    code,
			Message: err.Error(),
		},
	}
	respBytes, jsonErr := json.Marshal(resp)
	if jsonErr == nil {
		w.WriteHeader(500)
		w.Write(respBytes)
	} else {
		w.WriteHeader(500)
		log.Printf("error serializing failure response to json: %v\noriginal error: %v", jsonErr, err.Error())
		w.Write([]byte(`{"error":"internal server error"}`))
	}
}
