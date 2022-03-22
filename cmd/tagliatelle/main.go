package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"tagliatelle/pkg/settings"
	"tagliatelle/pkg/tagliatelle"
)

var (
	dryRun   bool
	filePath string
	pattern  string
	repo     string
	tag      string
)

func main() {
	if err := settings.Load(); err != nil {
		log.WithError(err).
			Error("failed to load settings")
	}

	flag.StringVar(&repo, "repo", "", "name of git repository")
	flag.StringVar(&filePath, "file", "", "file path to update")
	flag.StringVar(&tag, "tag", "", "new tag to use for update")
	flag.StringVar(&pattern, "pattern", "", "regex pattern to find and replace tag")
	flag.BoolVar(&dryRun, "dry-run", false, "enable dry run")
	flag.Parse()

	switch {
	case repo == "":
		invalid("repo")
	case filePath == "":
		invalid("filePath")
	case tag == "":
		invalid("tag")
	case pattern == "":
		invalid("pattern")
	}

	opts := tagliatelle.Options{
		DryRun:   dryRun,
		GitRepo:  repo,
		Pattern:  pattern,
		FilePath: filePath,
		Tag:      tag,
	}

	if err := tagliatelle.Entrypoint(opts); err != nil {
		log.WithError(err).
			Fatal("tagliatelle failed to run")
	}

	log.Info("finished")
}

func invalid(str string) {
	log.Fatal("invalid parameter: " + str)
}
