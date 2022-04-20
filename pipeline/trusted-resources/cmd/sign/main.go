/*
Copyright 2022 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/sigstore/cosign/cmd/cosign/cli/generate"
	"github.com/sigstore/cosign/pkg/signature"
	sigstore "github.com/sigstore/sigstore/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature/kms"
	"github.com/tektoncd/experimental/pipelines/trusted-resources/pkg/trustedtask"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/webhook/resourcesemantics"
	"sigs.k8s.io/yaml"
)

var (
	cosignKey    = flag.String("ck", "", "cosign private key path")
	kmsKey       = flag.String("kms", "", "kms key path")
	resourceFile = flag.String("rf", "", "YAML file path for tekton resources")
	// TODO: case insensitive for kind
	kind        = flag.String("kd", "Task", "The kind of the signed object. Supported values: [Task, Pipeline]")
	setdefaults = flag.Bool("sd", true, "Whether we add Tekton default values to the CRD before signing")
	targetDir   = flag.String("td", "", "Dir to save the signed files")
	targetFile  = flag.String("tf", "signed.yaml", "Filename of the signed file")
)

// This is a demo of how to generate signed task or pipeline files
func main() {
	ctx := context.Background()

	flag.Parse()

	var signer sigstore.Signer
	var err error
	if *cosignKey != "" {
		// Load signer from key files
		signer, err = signature.SignerFromKeyRef(ctx, *cosignKey, generate.GetPass)
		if err != nil {
			log.Fatalf("error getting signer: %v", err)
		}
	}
	if *kmsKey != "" {
		signer, err = kms.Get(ctx, *kmsKey, crypto.SHA256)
		if err != nil {
			log.Fatalf("error getting signer: %v", err)
		}
	}

	f, err := os.OpenFile(filepath.Join(*targetDir, *targetFile), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	defer f.Close()

	tsBuf, err := ioutil.ReadFile(*resourceFile)
	if err != nil {
		log.Fatalf("error reading file: %v", err)
	}
	var crd resourcesemantics.GenericCRD
	switch *kind {
	case "Task":
		crd = &v1beta1.Task{}
	case "Pipeline":
		crd = &v1beta1.Pipeline{}
	}

	if err := yaml.Unmarshal(tsBuf, &crd); err != nil {
		log.Fatalf("error unmarshalling Task/Pipeline: %v", err)
	}

	// Add missing fields for the crd
	if *setdefaults {
		crd.SetDefaults(ctx)
	}

	// Sign the task and write to writer
	if err := Sign(ctx, crd.(metav1.Object), signer, f); err != nil {
		log.Fatalf("error signing Task/Pipeline: %v", err)
	}

}

// Sign the crd and output signed bytes to writer
func Sign(ctx context.Context, o metav1.Object, signer sigstore.Signer, writer io.Writer) error {
	// get annotation
	a := o.GetAnnotations()
	if a == nil {
		a = map[string]string{}
	}

	// add signature
	sig, err := trustedtask.SignInterface(signer, o)
	if err != nil {
		return err
	}
	a[trustedtask.SignatureAnnotation] = sig
	o.SetAnnotations(a)
	signedBuf, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	_, err = writer.Write(signedBuf)
	return err
}
