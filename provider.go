// Copyright The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"

	"github.com/sigstore/cosign/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/pkg/cosign"
)

const (
	apiVersion = "externaldata.gatekeeper.sh/v1alpha1"
)

func main() {
	fmt.Println("starting server...")
	http.HandleFunc("/validate", validate)

	if err := http.ListenAndServe(":8090", nil); err != nil {
		panic(err)
	}
}

func validate(w http.ResponseWriter, req *http.Request) {
	// only accept POST requests
	if req.Method != http.MethodPost {
		sendResponse(nil, "only POST is allowed", w)
		return
	}

	// read request body
	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		sendResponse(nil, fmt.Sprintf("unable to read request body: %v", err), w)
		return
	}

	// parse request body
	var providerRequest externaldata.ProviderRequest
	err = json.Unmarshal(requestBody, &providerRequest)
	if err != nil {
		sendResponse(nil, fmt.Sprintf("unable to unmarshal request body: %v", err), w)
		return
	}

	results := make([]externaldata.Item, 0)

	ctx := req.Context()
	ro := options.RegistryOptions{}
	co, err := ro.ClientOpts(ctx)
	if err != nil {
		sendResponse(nil, fmt.Sprintf("ERROR: %v", err), w)
		return
	}

	// iterate over all keys
	for _, key := range providerRequest.Request.Keys {
		fmt.Println("verify signature for:", key)
		ref, err := name.ParseReference(key)
		if err != nil {
			sendResponse(nil, fmt.Sprintf("ERROR (ParseReference(%q)): %v", key, err), w)
			return
		}

		pemBytes, err := ioutil.ReadFile("/etc/ssl/certs/ca-certificates.crt")
		if err != nil {
			sendResponse(nil, fmt.Sprintf("ERROR: %v", err), w)
			return
		}

		rPool := x509.NewCertPool()
		rPool.AppendCertsFromPEM(pemBytes)

		checkedSignatures, _, err := cosign.VerifyImageSignatures(ctx, ref, &cosign.CheckOpts{
			RegistryClientOpts: co,
			RootCerts:          rPool,
		})

		if err != nil {
			fmt.Println(err)
			sendResponse(nil, fmt.Sprintf("VerifyImageSignatures: %v", err), w)
			return
		}

		if len(checkedSignatures) > 0 {
			fmt.Println("signature verified for:", key)
			fmt.Printf("%d number of valid signatures found for %s, found signatures: %v\n", len(checkedSignatures), key, checkedSignatures)
			results = append(results, externaldata.Item{
				Key:   key,
				Value: key + "_valid",
			})
		} else {
			fmt.Printf("no valid signatures found for: %s\n", key)
			results = append(results, externaldata.Item{
				Key:   key,
				Error: key + "_invalid",
			})
		}
	}

	sendResponse(&results, "", w)
}

// sendResponse sends back the response to Gatekeeper.
func sendResponse(results *[]externaldata.Item, systemErr string, w http.ResponseWriter) {
	response := externaldata.ProviderResponse{
		APIVersion: apiVersion,
		Kind:       "ProviderResponse",
	}

	if results != nil {
		response.Response.Items = *results
	} else {
		response.Response.SystemError = systemErr
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		panic(err)
	}
}
