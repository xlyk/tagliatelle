package tagliatelle

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"tagliatelle/pkg/settings"
	"time"

	memfs "github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rotisserie/eris"
	log "github.com/sirupsen/logrus"
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

//nolint:funlen // this function is expected to be long
func Entrypoint(o Options) error {
	// setup GitHub auth
	auth := &http.BasicAuth{
		Username: settings.GitUser,
		Password: settings.GitToken,
	}

	// clone git repo
	log.WithFields(log.Fields{
		"repo": o.GitRepo,
	}).Info("cloning git repo")

	r, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:  o.GitRepo,
		Auth: auth,
	})
	if err != nil {
		return eris.Wrap(err, "failed to clone repo")
	}

	// create worktree
	log.Info("creating worktree on filesystem")

	w, err := r.Worktree()
	if err != nil {
		return eris.Wrap(err, "failed to get repo worktree")
	}

	// get file contents
	log.WithFields(log.Fields{
		"file": o.FilePath,
	}).Info("reading file")

	data, err := readFile(o.FilePath)
	if err != nil {
		return eris.Wrap(err, "failed to read file")
	}

	// check for existing tag
	log.WithFields(log.Fields{
		"tag": o.Tag,
	}).Info("checking if tag already exists in file")

	oldTag, exists := checkTagAlreadyExists(data, o.Pattern, o.Tag)
	if exists {
		log.WithFields(log.Fields{
			"old": oldTag,
			"new": o.Tag,
		}).Warn("tag already exists in file... exiting early")

		return nil
	}

	log.WithFields(log.Fields{
		"old": oldTag,
		"new": o.Tag,
	}).Info("confirmed old tag != new tag")

	// use regex replace to update filePath with new tag
	log.WithFields(log.Fields{
		"pattern": o.Pattern,
	}).Info("replacing tag")

	modifiedData := regexReplace(data, o.Pattern, o.Tag)

	// write changes to file
	log.WithFields(log.Fields{
		"file": o.FilePath,
	}).Info("writing changes to file")

	if err := writeBytesToFile(o.FilePath, []byte(*modifiedData)); err != nil {
		return eris.Wrap(err, "failed to write file")
	}

	// git add filePath
	log.Info("adding file to index")

	_, err = w.Add(o.FilePath)
	if err != nil {
		return eris.Wrap(err, "failed to add file to index")
	}

	// set commit message
	msg := fmt.Sprintf("auto bump: %s", o.Tag)

	log.WithFields(log.Fields{
		"msg": msg,
	}).Info("setting commit message")

	hash, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name: "tagliatelle",
			When: time.Now().UTC(),
		},
	})
	if err != nil {
		return eris.Wrap(err, "failed to create commit")
	}

	// push the code to the remote
	log.WithFields(log.Fields{
		"hash": hash.String(),
	}).Info("pushing commit to remote")

	if o.DryRun {
		log.Info("dry-run successful - no changes made")
		fmt.Println("")
		fmt.Println(modifiedData)

		return nil
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return eris.Wrap(err, "failed to push commit to remote")
	}

	log.Info("remote repo successfully updated")

	return nil
}

func checkTagAlreadyExists(data *string, pattern, tag string) (string, bool) {
	var oldTag string

	m := regexp.MustCompile(pattern)

	res := m.FindAllStringSubmatch(*data, -1)

	if len(res) > 0 && len(res[0]) >= 2 {
		oldTag = res[0][2]
		if oldTag == tag {
			return oldTag, true
		}
	}

	return oldTag, false
}

func regexReplace(data *string, pattern, tag string) *string {
	m := regexp.MustCompile(pattern)

	t := fmt.Sprintf("${1}%s${3}", tag)

	res := m.ReplaceAllString(*data, t)

	return &res
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
