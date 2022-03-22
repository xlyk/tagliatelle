package tagliatelle

import (
	"bufio"
	"fmt"
	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
	"tagliatelle/pkg/settings"
)

type Options struct {
	DryRun   bool
	GitRepo  string
	Pattern  string
	Tag      string
	FilePath string
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

	// use regex replace to update filePath with new tag
	err = regexReplace(o.FilePath, o.Pattern, o.Tag, o.DryRun)
	if err != nil {
		return eris.Wrap(err, "failed to run regex replace")
	}

	if o.DryRun {
		return nil
	}

	// git add filePath
	_, err = w.Add(o.FilePath)
	if err != nil {
		return eris.Wrap(err, "failed to add file")
	}

	// set commit message
	msg := fmt.Sprintf("auto bump: %s", o.Tag)
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

func regexReplace(filename, pattern, tag string, dryRun bool) error {
	src, err := readFile(filename)
	if err != nil {
		return eris.Wrap(err, "failed to read file")
	}

	m := regexp.MustCompile(pattern)

	t := fmt.Sprintf("${1}%s${3}", tag)

	res := m.ReplaceAllString(*src, t)

	if dryRun {
		fmt.Println(res)
		return nil
	}

	// write changes to file
	if err := writeBytesToFile(filename, []byte(res)); err != nil {
		return eris.Wrap(err, "failed to write result to file")
	}

	return nil
}

func readFile(filename string) (*string, error) {
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

	return &allLines, nil
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
