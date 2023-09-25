package tagliatelle

import (
	"fmt"
	"tagliatelle/pkg/settings"
	"time"

	"github.com/pkg/errors"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	log "github.com/sirupsen/logrus"
)

type Repo struct {
	Auth     *http.BasicAuth
	Repo     *git.Repository
	Worktree *git.Worktree
	Options  Options
}

func NewRepo(o Options) (*Repo, error) {
	// setup GitHub auth
	auth := &http.BasicAuth{
		Username: settings.GitUser,
		Password: settings.GitToken,
	}

	return &Repo{
		Auth:    auth,
		Options: o,
	}, nil
}

func (r *Repo) updateFile(data *string) error {
	// use regex replace to update filePath with new tag
	log.WithFields(log.Fields{"pattern": r.Options.Pattern}).Info("replacing tag")

	modifiedData := regexReplace(data, r.Options.Pattern, r.Options.Tag)

	// write changes to file
	log.WithFields(log.Fields{"file": r.Options.FilePath}).Info("writing changes to file")

	if err := writeBytesToFile(r.Options.FilePath, []byte(*modifiedData)); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	// git add filePath
	log.Info("adding file to index")

	if _, err := r.Worktree.Add(r.Options.FilePath); err != nil {
		return errors.Wrap(err, "failed to add file to index")
	}

	// set commit message
	msg := fmt.Sprintf("auto bump: %s", r.Options.Tag)

	log.WithFields(log.Fields{
		"msg": msg,
	}).Info("setting commit message")

	parent, err := r.Repo.Head()
	if err != nil {
		return errors.Wrap(err, "failed to get repo HEAD")
	}

	hash, err := r.Worktree.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "tagliatelle",
			Email: "kylehanks@protonmail.com",
			When:  time.Now().UTC(),
		},
		Parents: []plumbing.Hash{parent.Hash()},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create commit")
	}

	// push the code to the remote
	log.WithFields(log.Fields{"hash": hash.String()}).Info("pushing commit to remote")

	if r.Options.DryRun {
		log.Info("dry-run successful - no changes made")

		return nil
	}

	if err = r.Repo.Push(&git.PushOptions{RemoteName: "origin", Auth: r.Auth}); err != nil {
		return errors.Wrap(err, "failed to push commit to remote")
	}

	log.Info("remote repo successfully updated")

	return nil
}

func (r *Repo) CheckoutMainBranch() error {
	// clone git repo
	log.WithFields(log.Fields{
		"repo": r.Options.GitRepo,
	}).Info("cloning git repo")

	repo, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:        r.Options.GitRepo,
		Auth:       r.Auth,
		RemoteName: "origin",
		NoCheckout: false,
		Depth:      1,
	})
	if err != nil {
		return errors.Wrap(err, "failed to clone repo")
	}

	// create worktree
	log.Info("creating worktree on filesystem")

	worktree, err := repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "failed to get repo worktree")
	}

	r.Repo = repo
	r.Worktree = worktree

	return nil
}
