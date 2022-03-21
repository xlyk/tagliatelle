package tagliatelle

import (
	"bufio"
	"fmt"
	"github.com/ghodss/yaml"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"
	"strings"
	"tagliatelle/pkg/settings"
)

type Options struct {
	GitRepo        string
	HelmPath       string
	KustomizeImage string
	Tag            string
	FilePath       string
	Mode           string
}

// create storage and filesystem
var (
	storage = memory.NewStorage()
	fs      = memfs.New()
)

func Entrypoint(o Options) error {

	// setup GitHub auth
	auth := &http.BasicAuth{
		Username: settings.GitUser,
		Password: settings.GitToken,
	}

	// clone git repo
	r, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:  o.GitRepo,
		Auth: auth,
	})
	if err != nil {
		return eris.Wrap(err, "failed to clone repo")
	}
	log.WithFields(log.Fields{
		"repo": o.GitRepo,
	}).Info("repo cloned")

	// create worktree
	w, err := r.Worktree()
	if err != nil {
		return eris.Wrap(err, "failed to get repo worktree")
	}

	// update filePath with new tag
	switch o.Mode {
	case "helm":
		if err := updateHelmValues(o.FilePath, o.Tag, o.HelmPath); err != nil {
			return eris.Wrap(err, "failed to update helm values")
		}
	case "kustomize":
		if err := updateKustomizationImage(o.FilePath, o.Tag, o.KustomizeImage); err != nil {
			return eris.Wrap(err, "failed to update helm values")
		}
	default:
		return eris.Wrap(err, "invalid mode")
	}

	// git add filePath
	_, err = w.Add(o.FilePath)
	if err != nil {
		return eris.Wrap(err, "failed to add file")
	}

	// set commit message
	msg := fmt.Sprintf("auto bump %s:%s", o.KustomizeImage, o.Tag)
	_, err = w.Commit(msg, &git.CommitOptions{})
	if err != nil {
		return eris.Wrap(err, "failed to create commit")
	}

	// push the code to the remote
	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return eris.Wrap(err, "failed to push commit to origin")
	}

	log.Info("remote repo updated")
	return nil
}

func updateHelmValues(filePath, tag, path string) error {
	return eris.New("not implemented")
}

type KustomizeImage struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	NewName string `json:"newName,omitempty" yaml:"newName,omitempty"`
	NewTag  string `json:"newTag,omitempty" yaml:"newTag,omitempty"`
}

func getKustomizeImage(img interface{}) (*KustomizeImage, error) {
	yamlBytes, err := yaml.Marshal(&img)
	if err != nil {
		return nil, eris.Wrap(err, "failed to marshal yaml")
	}

	var image KustomizeImage
	err = yaml.Unmarshal(yamlBytes, &image)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal yaml")
	}

	return &image, nil
}

func updateKustomizationImage(filePath, tag, image string) error {
	kustomizeMap, err := readFile(filePath)
	if err != nil {
		return eris.Wrap(err, "failed to read file")
	}

	updated := false

	// iterate existing images and populate new slice
	var kustomizeImages []KustomizeImage
	for _, img := range kustomizeMap["images"].([]interface{}) {
		// get KustomizeImage
		kImage, err := getKustomizeImage(img)
		if err != nil {
			log.Error("failed to get kustomize image")
			continue
		}

		// compare image value here, update if matches
		if kImage.Name == image {
			kImage.NewTag = tag
			updated = true
		}

		// add image to slice
		kustomizeImages = append(kustomizeImages, *kImage)
	}

	// if no match then error
	if !updated {
		return eris.New("no match found, failed to update image")
	}

	// replace images in kustomizeMap w/ kustomizeImages
	kustomizeMap["images"] = kustomizeImages

	// marshal yaml
	yamlBytes, err := yaml.Marshal(&kustomizeMap)
	if err != nil {
		return eris.Wrap(err, "failed to marshal yaml")
	}

	// write kustomizeMap to yaml file
	err = writeBytesToFile(filePath, yamlBytes)
	if err != nil {
		return eris.Wrap(err, "failed to write bytes to file")
	}

	return nil
}

func readFile(filename string) (map[string]interface{}, error) {
	var output map[string]interface{}

	f, err := fs.Open(filename)
	if err != nil {
		return nil, eris.Wrap(err, "failed to read file")
	}

	var lines []string

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, eris.Wrap(err, "failed to read file with scanner")
	}

	allLines := strings.Join(lines, "\n")

	err = yaml.Unmarshal([]byte(allLines), &output)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal yaml")
	}

	return output, nil
}

func writeBytesToFile(outputFile string, data []byte) error {
	f, err := fs.Create(outputFile)
	if err != nil {
		return eris.Wrap(err, "failed to open file to write data to")
	}

	_, err = f.Write(data)
	if err != nil {
		return eris.Wrap(err, "failed to write data to file")
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.WithError(err).Error("failed to close file")
		}
	}()

	return nil
}
