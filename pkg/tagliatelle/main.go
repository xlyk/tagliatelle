package tagliatelle

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/go-git/go-billy/v5/memfs"
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

func Entrypoint(o Options) error {
	repo, err := NewRepo(o)
	if err != nil {
		return errors.Wrap(err, "failed to create Repo struct")
	}

	if err := repo.CheckoutMainBranch(); err != nil {
		return errors.Wrap(err, "failed to clone repository")
	}

	data, err := readFile(o.FilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}

	exists, err := checkTagAlreadyExists(data, o.Pattern, o.Tag)
	if err != nil {
		return errors.Wrap(err, "failed to check if tag already exists")
	}

	if exists {
		return nil
	}

	if err = repo.updateFile(data); err != nil {
		return errors.Wrap(err, "failed to update file")
	}

	return nil
}

func checkTagAlreadyExists(data *string, pattern, tag string) (bool, error) {
	log.WithFields(log.Fields{
		"tag": tag,
	}).Info("checking if tag already exists in file")

	var oldTag string

	m := regexp.MustCompile(pattern)

	res := m.FindAllStringSubmatch(*data, -1)

	if len(res) > 0 && len(res[0]) >= 2 {
		oldTag = res[0][2]
		if oldTag == tag {
			log.WithFields(log.Fields{
				"old": oldTag,
				"new": tag,
			}).Warn("tag already exists in file... exiting early")

			return true, nil
		}
	}

	if oldTag == "" {
		return false, eris.New("failed to find old tag")
	}

	log.WithFields(log.Fields{
		"old": oldTag,
		"new": tag,
	}).Info("confirmed old tag != new tag")

	return false, nil
}

func regexReplace(data *string, pattern, tag string) *string {
	m := regexp.MustCompile(pattern)

	t := fmt.Sprintf("${1}%s${3}", tag)

	res := m.ReplaceAllString(*data, t)

	return &res
}

func readFile(filename string) (*string, error) {
	// get file contents
	log.WithFields(log.Fields{
		"file": filename,
	}).Info("reading file")

	f, err := fs.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}

	var lines []string

	// read the file line by line using scanner
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read file with scanner")
	}

	allLines := strings.Join(lines, "\n")

	return &allLines, nil
}

func writeBytesToFile(outputFile string, data []byte) error {
	f, err := fs.Create(outputFile)
	if err != nil {
		return errors.Wrap(err, "failed to open file to write data to")
	}

	_, err = f.Write(data)
	if err != nil {
		return errors.Wrap(err, "failed to write data to file")
	}

	defer func() {
		err := f.Close()
		if err != nil {
			log.WithError(err).Error("failed to close file")
		}
	}()

	return nil
}
