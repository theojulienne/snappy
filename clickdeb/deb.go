package clickdeb

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"launchpad.net/snappy/helpers"

	"github.com/blakesmith/ar"
)

var (
	// ErrSnapInvalidContent is returned if a snap package contains
	// invalid content
	ErrSnapInvalidContent = errors.New("snap contains invalid content")
)

func xzPipeReader(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	cmd := exec.Command("xz", "--decompress", "--stdout")
	cmd.Stdin = r
	cmd.Stdout = pw

	// run xz in its own go-routine
	go func() {
		pw.CloseWithError(cmd.Run())
	}()

	return pr
}

func clickVerifyContentFn(path string) (string, error) {
	path = filepath.Clean(path)
	if strings.Contains(path, "..") {
		return "", ErrSnapInvalidContent
	}

	return path, nil
}

// ClickDeb provides support for the "click" containers (a special kind of
// deb package)
type ClickDeb struct {
	Path string

	file *os.File
}

// ControlContent returns the content of the given control data member
// (e.g. the content of the "manifest" file in the control.tar.gz ar member)
func (d *ClickDeb) ControlContent(controlMember string) ([]byte, error) {
	var err error

	d.file, err = os.Open(d.Path)
	if err != nil {
		return nil, err
	}
	defer d.file.Close()

	dataReader, err := d.skipToArMember("control.tar")
	if err != nil {
		return nil, err
	}

	var content []byte
	err = helpers.TarIterate(dataReader, func(tr *tar.Reader, hdr *tar.Header) error {
		if filepath.Clean(hdr.Name) == controlMember {
			content, err = ioutil.ReadAll(tr)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return content, nil
}

// Unpack unpacks the data.tar.{gz,bz2,xz} into the given target directory
// with click specific verification, i.e. no files will be extracted outside
// of the targetdir (no ".." inside the data.tar is allowed)
func (d *ClickDeb) Unpack(targetDir string) error {
	var err error

	d.file, err = os.Open(d.Path)
	if err != nil {
		return err
	}
	defer d.file.Close()

	dataReader, err := d.skipToArMember("data.tar")
	if err != nil {
		return err
	}

	// and unpack
	return helpers.UnpackTar(dataReader, targetDir, clickVerifyContentFn)
}

func (d *ClickDeb) skipToArMember(memberPrefix string) (io.Reader, error) {
	var err error

	// find the right ar member
	arReader := ar.NewReader(d.file)
	var header *ar.Header
	for {
		header, err = arReader.Next()
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(header.Name, memberPrefix) {
			break
		}
	}

	// find out what compression
	var dataReader io.Reader
	switch {
	case strings.HasSuffix(header.Name, ".gz"):
		dataReader, err = gzip.NewReader(d.file)
		if err != nil {
			return nil, err
		}
	case strings.HasSuffix(header.Name, ".bz2"):
		dataReader = bzip2.NewReader(d.file)
	case strings.HasSuffix(header.Name, ".xz"):
		dataReader = xzPipeReader(d.file)
	default:
		return nil, fmt.Errorf("Can not handle %s", header.Name)
	}

	return dataReader, nil
}