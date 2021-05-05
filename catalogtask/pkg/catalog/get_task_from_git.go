package catalog

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
)

// "git-clone" = "git-clone/0.3/git-clone" (always use latest version)
// "git-clone--0.2" = "git-clone/0.2/git-clone"
func (c *Catalog) Get(name string) (*v1beta1.Task, error) {
	version := ""

	if strings.Contains(name, "--") {
		s := strings.Split(name, "--")
		name = s[0]
		version = s[1]
	} else {
		root := filepath.Join(c.dir, fmt.Sprintf("/%s/%s", c.kind, name))
		ver, err := c.findLatestVersion(root)
		if err != nil {
			return nil, fmt.Errorf("error scanning %q: %v", root, err)
		}
		version = ver
	}

	path := fmt.Sprintf("/%s/%s/%s/%s.yaml", c.kind, name, version, name)

	return c.readResourceFromFile(path)
}

func (c *Catalog) readResourceFromFile(path string) (*v1beta1.Task, error) {
	yamlPath := filepath.Join(c.dir, path)
	yamlBytes, err := ioutil.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml in %q: %v", path, err)
	}

	decoder := scheme.Codecs.UniversalDeserializer()
	obj, _, err := decoder.Decode(yamlBytes, nil, nil)

	switch t := obj.(type) {
	case *v1beta1.Task:
		return t, nil
	default:
		return nil, fmt.Errorf("%q not a %s: %v", path, c.kind, err)
	}
}

func (c *Catalog) cloneRepo() error {
	cmd := exec.Command("git", "clone", c.repoURL, c.dir)
	gitLogs, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error cloning repo %q: %v", c.repoURL, err)
	}
	log.Println(string(gitLogs))
	return nil
}

// findLatestVersion finds the latest catalog entry version from a entry's root directory
// Given a root directory structed as root/0.3, root/0.2, root/0.1 this function should return "0.3"
func (c *Catalog) findLatestVersion(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	names := []string{}
	for _, ent := range entries {
		if strings.HasPrefix(ent.Name(), ".") {
			continue
		}
		if ent.IsDir() {
			names = append(names, ent.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no versions found in root %v", root)
	}

	// TODO(sbwsg): sort by semver instead of just alphabetical
	sort.Strings(names)

	return names[len(names)-1], nil
}
