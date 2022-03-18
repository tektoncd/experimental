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
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/sigstore/cosign/cmd/cosign/cli/generate"
	"github.com/sigstore/cosign/pkg/signature"
	sigstore "github.com/sigstore/sigstore/pkg/signature"
	"github.com/tektoncd/experimental/pipelines/trusted-resources/pkg/trustedtask"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

var (
	privateKey  = flag.String("pk", "", "cosign private key path")
	taskRunFile = flag.String("tr", "", "YAML file path for tekton taskrun")
	targetDir   = flag.String("td", "", "Dir to save the signed files")
	targetFile  = flag.String("tf", "signed.yaml", "Filename of the signed file")
)

// This is a demo of how to generate signed taskrun files, task file is not supported yet.
func main() {
	ctx := context.Background()

	flag.Parse()

	// Load signer from key files
	signer, err := signature.SignerFromKeyRef(ctx, *privateKey, generate.GetPass)
	if err != nil {
		log.Fatalf("error getting signer: %v", err)
	}

	f, err := os.OpenFile(filepath.Join(*targetDir, *targetFile), os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	defer f.Close()

	// Read taskrun objects from yaml files
	trBuf, err := ioutil.ReadFile(*taskRunFile)
	if err != nil {
		log.Fatalf("error reading taskrun: %v", err)
	}

	tr := &v1beta1.TaskRun{}
	if err := yaml.Unmarshal(trBuf, &tr); err != nil {
		log.Fatalf("error unmarshalling taskrun: %v", err)
	}

	// Sign the object and write to writer
	if err := SignTaskRun(ctx, tr, signer, f); err != nil {
		log.Fatalf("error signing taskrun: %v", err)
	}

}

// Sign the taskrun and output signed bytes to writer
func SignTaskRun(ctx context.Context, tr *v1beta1.TaskRun, signer sigstore.Signer, writer io.Writer) error {
	sig, err := trustedtask.SignInterface(signer, tr)
	if err != nil {
		return err
	}

	if tr.Annotations == nil {
		tr.Annotations = map[string]string{trustedtask.SignatureAnnotation: sig}
	} else {
		tr.Annotations[trustedtask.SignatureAnnotation] = sig
	}

	signedBuf, err := yaml.Marshal(tr)
	if err != nil {
		return err
	}

	_, err = writer.Write(signedBuf)
	return err
}
