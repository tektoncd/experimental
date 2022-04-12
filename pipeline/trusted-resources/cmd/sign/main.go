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
	"sigs.k8s.io/yaml"
)

var (
	cosignKey  = flag.String("ck", "", "cosign private key path")
	kmsKey     = flag.String("kms", "", "kms key path")
	taskFile   = flag.String("ts", "", "YAML file path for tekton task")
	targetDir  = flag.String("td", "", "Dir to save the signed files")
	targetFile = flag.String("tf", "signed.yaml", "Filename of the signed file")
)

// This is a demo of how to generate signed task files
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

	tsBuf, err := ioutil.ReadFile(*taskFile)
	if err != nil {
		log.Fatalf("error reading task: %v", err)
	}

	ts := &v1beta1.Task{}
	if err := yaml.Unmarshal(tsBuf, &ts); err != nil {
		log.Fatalf("error unmarshalling taskrun: %v", err)
	}

	// Sign the task and write to writer
	if err := SignTask(ctx, ts, signer, f); err != nil {
		log.Fatalf("error signing taskrun: %v", err)
	}

}

// Sign the taskrun and output signed bytes to writer
func SignTask(ctx context.Context, task *v1beta1.Task, signer sigstore.Signer, writer io.Writer) error {
	sig, err := trustedtask.SignInterface(signer, task)
	if err != nil {
		return err
	}

	if task.Annotations == nil {
		task.Annotations = map[string]string{}
	}
	task.Annotations[trustedtask.SignatureAnnotation] = sig

	signedBuf, err := yaml.Marshal(task)
	if err != nil {
		return err
	}

	_, err = writer.Write(signedBuf)
	return err
}
