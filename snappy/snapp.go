package snappy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	yaml "launchpad.net/goyaml"
)

// snapPart represents a generic snap type
type snapPart struct {
	name        string
	version     string
	description string
	hash        string
	isActive    bool
	isInstalled bool
	stype       SnapType

	basedir string
}

type packageYaml struct {
	Name    string
	Version string
	Vendor  string
	Icon    string
	Type    SnapType
}

type remoteSnap struct {
	Publisher       string  `json:"publisher,omitempty"`
	Name            string  `json:"name"`
	Title           string  `json:"title"`
	IconURL         string  `json:"icon_url"`
	Price           float64 `json:"price,omitempty"`
	Content         string  `json:"content,omitempty"`
	RatingsAverage  float64 `json:"ratings_average,omitempty"`
	Version         string  `json:"version"`
	AnonDownloadURL string  `json:"anon_download_url, omitempty"`
	DownloadURL     string  `json:"download_url, omitempty"`
	DownloadSha512  string  `json:"download_sha512, omitempty"`
}

type searchResults struct {
	Payload struct {
		Packages []remoteSnap `json:"clickindex:package"`
	} `json:"_embedded"`
}

// NewInstalledSnapPart returns a new snapPart from the given yamlPath
func newInstalledSnapPart(yamlPath string) *snapPart {
	part := snapPart{}

	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		return nil
	}

	r, err := os.Open(yamlPath)
	if err != nil {
		log.Printf("Can not open '%s'", yamlPath)
		return nil
	}

	yamlData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Printf("Can not read '%v'", r)
		return nil
	}

	var m packageYaml
	err = yaml.Unmarshal(yamlData, &m)
	if err != nil {
		log.Printf("Can not parse '%s'", yamlData)
		return nil
	}
	part.basedir = filepath.Dir(filepath.Dir(yamlPath))
	// data from the yaml
	part.name = m.Name
	part.version = m.Version
	part.isInstalled = true
	// check if the part is active
	allVersionsDir := filepath.Dir(part.basedir)
	p, _ := filepath.EvalSymlinks(filepath.Join(allVersionsDir, "current"))
	if p == part.basedir {
		part.isActive = true
	}
	part.stype = m.Type

	return &part
}

// Type returns the type of the snapPart (app, oem, ...)
func (s *snapPart) Type() SnapType {
	if s.stype != "" {
		return s.stype
	}
	// if not declared its a app
	return "app"
}

// Name returns the name
func (s *snapPart) Name() string {
	return s.name
}

// Version returns the version
func (s *snapPart) Version() string {
	return s.version
}

// Description returns the description
func (s *snapPart) Description() string {
	return s.description
}

// Channel returns the channel
func (s *snapPart) Channel() string {
	// FIXME: hardcoded
	return "edge"
}

// Hash returns the hash
func (s *snapPart) Hash() string {
	return s.hash
}

// IsActive returns true if the snap is active
func (s *snapPart) IsActive() bool {
	return s.isActive
}

// IsInstalled returns true if the snap is installed
func (s *snapPart) IsInstalled() bool {
	return s.isInstalled
}

// InstalledSize returns the size of the installed snap
func (s *snapPart) InstalledSize() int {
	return -1
}

// DownloadSize returns the dowload size
func (s *snapPart) DownloadSize() int {
	return -1
}

// Install installs the snap
func (s *snapPart) Install(pb ProgressMeter) (err error) {
	return errors.New("Install of a local part is not possible")
}

// SetActive sets the snap active
func (s *snapPart) SetActive() (err error) {
	return setActiveClick(s.basedir)
}

// Uninstall remove the snap from the system
func (s *snapPart) Uninstall() (err error) {
	err = removeClick(s.basedir)
	return err
}

// Config is used to to configure the snap
func (s *snapPart) Config(configuration []byte) (err error) {
	return err
}

// NeedsReboot returns true if the snap becomes active on the next reboot
func (s *snapPart) NeedsReboot() bool {
	return false
}

// SnapLocalRepository is the type for a local snap repository
type snapLocalRepository struct {
	path string
}

// newLocalSnapRepository returns a new snapLocalRepository for the given
// path
func newLocalSnapRepository(path string) *snapLocalRepository {
	if s, err := os.Stat(path); err != nil || !s.IsDir() {
		return nil
	}
	return &snapLocalRepository{path: path}
}

// Description describes the local repository
func (s *snapLocalRepository) Description() string {
	return fmt.Sprintf("Snap local repository for %s", s.path)
}

// Search searches the local repository
func (s *snapLocalRepository) Search(terms string) (versions []Part, err error) {
	return versions, err
}

// Details returns details for the given snap
func (s *snapLocalRepository) Details(terms string) (versions []Part, err error) {
	return versions, err
}

// Updates returns the available updates
func (s *snapLocalRepository) Updates() (parts []Part, err error) {
	return parts, err
}

// Installed returns the installed snaps from this repository
func (s *snapLocalRepository) Installed() (parts []Part, err error) {
	globExpr := filepath.Join(s.path, "*", "*", "meta", "package.yaml")
	matches, err := filepath.Glob(globExpr)
	if err != nil {
		return parts, err
	}
	for _, yamlfile := range matches {

		// skip "current" and similar symlinks
		realpath, err := filepath.EvalSymlinks(yamlfile)
		if err != nil {
			return parts, err
		}
		if realpath != yamlfile {
			continue
		}

		snap := newInstalledSnapPart(yamlfile)
		if snap != nil {
			parts = append(parts, snap)
		}
	}

	return parts, err
}

// remoteSnapPart represents a snap available on the server
type remoteSnapPart struct {
	pkg remoteSnap
}

// Type returns the type of the snapPart (app, oem, ...)
func (s *remoteSnapPart) Type() SnapType {
	// FIXME: the store does not publish this info
	return SnapTypeApp
}

// Name returns the name
func (s *remoteSnapPart) Name() string {
	return s.pkg.Name
}

// Version returns the version
func (s *remoteSnapPart) Version() string {
	return s.pkg.Version
}

// Description returns the description
func (s *remoteSnapPart) Description() string {
	return s.pkg.Title
}

// Channel returns the channel
func (s *remoteSnapPart) Channel() string {
	// FIXME: hardcoded
	return "edge"
}

// Hash returns the hash
func (s *remoteSnapPart) Hash() string {
	return "FIXME"
}

// IsActive returns true if the snap is active
func (s *remoteSnapPart) IsActive() bool {
	return false
}

// IsInstalled returns true if the snap is installed
func (s *remoteSnapPart) IsInstalled() bool {
	return false
}

// InstalledSize returns the size of the installed snap
func (s *remoteSnapPart) InstalledSize() int {
	return -1
}

// DownloadSize returns the dowload size
func (s *remoteSnapPart) DownloadSize() int {
	return -1
}

// Install installs the snap
func (s *remoteSnapPart) Install(pbar ProgressMeter) (err error) {
	w, err := ioutil.TempFile("", s.pkg.Name)
	if err != nil {
		return err
	}
	defer func() {
		w.Close()
		os.Remove(w.Name())
	}()

	resp, err := http.Get(s.pkg.AnonDownloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if pbar != nil {
		pbar.Start(float64(resp.ContentLength))
		mw := io.MultiWriter(w, pbar)
		_, err = io.Copy(mw, resp.Body)
		pbar.Finished()
	} else {
		_, err = io.Copy(w, resp.Body)
	}

	if err != nil {
		return err
	}

	err = installClick(w.Name(), 0)
	if err != nil {
		return err
	}

	return err
}

// SetActive sets the snap active
func (s *remoteSnapPart) SetActive() (err error) {
	return errors.New("A remote part must be installed first")
}

// Uninstall remove the snap from the system
func (s *remoteSnapPart) Uninstall() (err error) {
	return errors.New("Uninstall of a remote part is not possible")
}

// Config is used to to configure the snap
func (s *remoteSnapPart) Config(configuration []byte) (err error) {
	return err
}

// NeedsReboot returns true if the snap becomes active on the next reboot
func (s *remoteSnapPart) NeedsReboot() bool {
	return false
}

// newRemoteSnapPart returns a new remoteSnapPart from the given
// remoteSnap data
func newRemoteSnapPart(data remoteSnap) *remoteSnapPart {
	return &remoteSnapPart{pkg: data}
}

// snapUbuntuStoreRepository represents the ubuntu snap store
type snapUbuntuStoreRepository struct {
	searchURI  string
	detailsURI string
	bulkURI    string
}

// NewUbuntuStoreSnapRepository creates a new snapUbuntuStoreRepository
func newUbuntuStoreSnapRepository() *snapUbuntuStoreRepository {
	return &snapUbuntuStoreRepository{
		searchURI:  "https://search.apps.ubuntu.com/api/v1/search?q=%s",
		detailsURI: "https://search.apps.ubuntu.com/api/v1/package/%s",
		bulkURI:    "https://myapps.developer.ubuntu.com/dev/api/click-metadata/"}
}

// Description describes the repository
func (s *snapUbuntuStoreRepository) Description() string {
	return fmt.Sprintf("Snap remote repository for %s", s.searchURI)
}

// Details returns details for the given snap in this repository
func (s *snapUbuntuStoreRepository) Details(snapName string) (parts []Part, err error) {
	url := fmt.Sprintf(s.detailsURI, snapName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return parts, err
	}

	// set headers
	req.Header.Set("Accept", "application/hal+json")
	frameworks, _ := InstalledSnapNamesByType(SnapTypeFramework)
	frameworks = append(frameworks, "ubuntu-core-15.04-dev1")
	req.Header.Set("X-Ubuntu-Frameworks", strings.Join(frameworks, ","))
	req.Header.Set("X-Ubuntu-Architecture", Architecture())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return parts, err
	}
	defer resp.Body.Close()

	// check statusCode
	switch {
	case resp.StatusCode == 404:
		return parts, ErrRemoteSnapNotFound
	case resp.StatusCode != 200:
		return parts, fmt.Errorf("snapUbuntuStoreRepository: unexpected http statusCode %v for %s", resp.StatusCode, snapName)
	}

	// and decode json
	var detailsData remoteSnap
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&detailsData); err != nil {
		return nil, err
	}

	snap := newRemoteSnapPart(detailsData)
	parts = append(parts, snap)

	return parts, err
}

// Search searches the repository for the given searchTerm
func (s *snapUbuntuStoreRepository) Search(searchTerm string) (parts []Part, err error) {
	url := fmt.Sprintf(s.searchURI, searchTerm)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return parts, err
	}

	// set headers
	req.Header.Set("Accept", "application/hal+json")
	frameworks, _ := InstalledSnapNamesByType(SnapTypeFramework)
	frameworks = append(frameworks, "ubuntu-core-15.04-dev1")
	req.Header.Set("X-Ubuntu-Frameworks", strings.Join(frameworks, ","))
	req.Header.Set("X-Ubuntu-Architecture", Architecture())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return parts, err
	}
	defer resp.Body.Close()

	var searchData searchResults

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&searchData); err != nil {
		return nil, err
	}

	for _, pkg := range searchData.Payload.Packages {
		snap := newRemoteSnapPart(pkg)
		parts = append(parts, snap)
	}

	return parts, err
}

// Updates returns the available updates
func (s *snapUbuntuStoreRepository) Updates() (parts []Part, err error) {
	// the store only supports apps and framworks currently, so no
	// sense in sending it our ubuntu-core snap
	installed, err := InstalledSnapNamesByType(SnapTypeApp, SnapTypeFramework)
	if err != nil || len(installed) == 0 {
		return parts, err
	}
	jsonData, err := json.Marshal(map[string][]string{"name": installed})
	if err != nil {
		return parts, err
	}

	req, err := http.NewRequest("POST", s.bulkURI, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateData []remoteSnap
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&updateData); err != nil {
		return nil, err
	}

	for _, pkg := range updateData {
		snap := newRemoteSnapPart(pkg)
		parts = append(parts, snap)
	}

	return parts, nil
}

// Installed returns the installed snaps from this repository
func (s *snapUbuntuStoreRepository) Installed() (parts []Part, err error) {
	return parts, err
}
