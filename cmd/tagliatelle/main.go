package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"tagliatelle/pkg/settings"
	"tagliatelle/pkg/tagliatelle"
)

func main() {
	if err := settings.Load(); err != nil {
		log.WithError(err).Error("failed to load settings")
	}

	var (
		filePath       string
		helmPath       string
		kustomizeImage string
		mode           string
		repo           string
		tag            string
	)

	flag.StringVar(&repo, "repo", "", "name of git repository")
	flag.StringVar(&filePath, "file", "", "file path to update")
	flag.StringVar(&tag, "tag", "", "new tag to use for update")
	flag.StringVar(&mode, "mode", "", "mode to use [helm|kustomize]")
	flag.StringVar(&helmPath, "helm-path", "", "path to update in helm values file")
	flag.StringVar(&kustomizeImage, "kustomize-image", "", "name of image in kustomization file")
	flag.Parse()

	if repo == "" {
		log.Error("invalid repo")
		os.Exit(1)
		return
	}

	if filePath == "" {
		log.Error("invalid file")
		os.Exit(1)
		return
	}

	if tag == "" {
		log.Error("invalid tag")
		os.Exit(1)
		return
	}

	switch mode {
	case "helm":
		if helmPath == "" {
			log.Error("invalid helm-path")
			os.Exit(1)
			return
		}
		break
	case "kustomize":
		if kustomizeImage == "" {
			log.Error("invalid kustomize-image")
			os.Exit(1)
			return
		}
		break
	default:
		log.Error("invalid mode")
		os.Exit(1)
		return
	}

	opts := tagliatelle.Options{
		GitRepo:        repo,
		HelmPath:       helmPath,
		KustomizeImage: kustomizeImage,
		FilePath:       filePath,
		Tag:            tag,
		Mode:           mode,
	}

	if err := tagliatelle.Entrypoint(opts); err != nil {
		log.Error("tagliatelle failed to run")
		os.Exit(1)
		return
	}

	log.Info("finished")
	os.Exit(0)
}
