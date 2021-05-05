package catalog

import "io/ioutil"

type Catalog struct {
	kind    string
	repoURL string
	// dir is where the repo is checked out
	dir string
}

func New(repoURL, kind, scratchDir string) (Catalog, error) {
	catalog := Catalog{}
	dir, err := ioutil.TempDir(scratchDir, "repo")
	if err != nil {
		return catalog, err
	}
	catalog.repoURL = repoURL
	// TODO: check this is valid (either "pipeline" or "task")
	catalog.kind = kind
	catalog.dir = dir
	if err := catalog.cloneRepo(); err != nil {
		return Catalog{}, err
	} else {
		return catalog, nil
	}
}
