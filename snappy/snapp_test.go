package snappy

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	partition "launchpad.net/snappy/partition"

	. "launchpad.net/gocheck"
)

const (
	packageHello = `
name: hello-app
version: 1.10
vendor: Michael Vogt <mvo@ubuntu.com>
icon: meta/hello.svg
binaries:
 - name: bin/hello
`
)

type SnapTestSuite struct {
	tempdir string
}

var _ = Suite(&SnapTestSuite{})

func (s *SnapTestSuite) SetUpTest(c *C) {
	s.tempdir = c.MkDir()
	newPartition = func() (p partition.Interface) {
		return new(MockPartition)
	}

	snapDataDir = filepath.Join(s.tempdir, "/var/lib/apps/")
	snapAppsDir = filepath.Join(s.tempdir, "/apps/")
	snapOemDir = filepath.Join(s.tempdir, "/oem/")

	// we may not have debsig-verify installed (and we don't need it
	// for the unittests)
	runDebsigVerify = func(snapFile string, allowUnauth bool) (err error) {
		return nil
	}
}

func (s *SnapTestSuite) makeMockSnap() (snapDir string, err error) {
	metaDir := filepath.Join(s.tempdir, "apps", "hello-app", "1.10", "meta")
	err = os.MkdirAll(metaDir, 0777)
	if err != nil {
		return "", err
	}
	yamlFile := filepath.Join(metaDir, "package.yaml")
	ioutil.WriteFile(yamlFile, []byte(packageHello), 0666)

	snapDir, _ = filepath.Split(metaDir)
	return yamlFile, err
}

func makeSnapActive(packageYamlPath string) (err error) {
	snapdir := filepath.Dir(filepath.Dir(packageYamlPath))
	parent := filepath.Dir(snapdir)
	err = os.Symlink(snapdir, filepath.Join(parent, "current"))

	return err
}

func (s *SnapTestSuite) TestLocalSnapInvalidPath(c *C) {
	snap := newInstalledSnapPart("invalid-path")
	c.Assert(snap, IsNil)
}

func (s *SnapTestSuite) TestLocalSnapSimple(c *C) {
	snapYaml, err := s.makeMockSnap()
	c.Assert(err, IsNil)

	snap := newInstalledSnapPart(snapYaml)
	c.Assert(snap, NotNil)
	c.Assert(snap.Name(), Equals, "hello-app")
	c.Assert(snap.Version(), Equals, "1.10")
	c.Assert(snap.IsActive(), Equals, false)

	c.Assert(snap.basedir, Equals, filepath.Join(s.tempdir, "apps", "hello-app", "1.10"))
}

func (s *SnapTestSuite) TestLocalSnapActive(c *C) {
	snapYaml, err := s.makeMockSnap()
	c.Assert(err, IsNil)
	makeSnapActive(snapYaml)

	snap := newInstalledSnapPart(snapYaml)
	c.Assert(snap.IsActive(), Equals, true)
}

func (s *SnapTestSuite) TestLocalSnapRepositoryInvalid(c *C) {
	snap := newLocalSnapRepository("invalid-path")
	c.Assert(snap, IsNil)
}

func (s *SnapTestSuite) TestLocalSnapRepositorySimple(c *C) {
	yamlPath, err := s.makeMockSnap()
	c.Assert(err, IsNil)
	err = makeSnapActive(yamlPath)
	c.Assert(err, IsNil)

	snap := newLocalSnapRepository(filepath.Join(s.tempdir, "apps"))
	c.Assert(snap, NotNil)

	installed, err := snap.Installed()
	c.Assert(err, IsNil)
	c.Assert(len(installed), Equals, 1)
	c.Assert(installed[0].Name(), Equals, "hello-app")
	c.Assert(installed[0].Version(), Equals, "1.10")
}

/* acquired via:
   curl  -H 'accept: application/hal+json' -H "X-Ubuntu-Frameworks: ubuntu-core-15.04-dev1" -H "X-Ubuntu-Architecture: amd64" https://search.apps.ubuntu.com/api/v1/search?q=hello
*/
const MockSearchJSON = `{
  "_links": {
    "self": {
      "href": "https:\/\/search.apps.ubuntu.com\/api\/v1\/search?q=xkcd"
    },
    "curies": [
      {
        "templated": true,
        "name": "clickindex",
        "href": "https:\/\/search.apps.ubuntu.com\/docs\/relations.html{#rel}"
      }
    ]
  },
  "_embedded": {
    "clickindex:package": [
      {
        "prices": null,
        "_links": {
          "self": {
            "href": "https:\/\/search.apps.ubuntu.com\/api\/v1\/package\/com.ubuntu.snappy.xkcd-webserver"
          }
        },
        "version": "0.1",
        "ratings_average": 0.0,
        "content": "application",
        "price": 0.0,
        "icon_url": "https:\/\/myapps.developer.ubuntu.com\/site_media\/appmedia\/2014\/12\/xkcd.svg.png",
        "title": "Show random XKCD comic",
        "name": "xkcd-webserver.mvo",
        "publisher": "Canonical"
      }
    ]
  }
}`

/* acquired via:
curl --data-binary '{"name":["docker","foo","com.ubuntu.snappy.hello-world","httpd-minimal-golang-example","owncloud","xkcd-webserver"]}'  -H 'content-type: application/json' https://myapps.developer.ubuntu.com/dev/api/click-metadata/
*/
const MockUpdatesJSON = `
[
    {
        "status": "Published",
        "name": "hello-world",
        "changelog": "",
        "icon_url": "https://myapps.developer.ubuntu.com/site_media/appmedia/2015/01/hello.svg.png",
        "title": "Hello world example",
        "binary_filesize": 31166,
        "anon_download_url": "https://public.apps.ubuntu.com/anon/download/com.ubuntu.snappy/hello-world/hello-world_1.0.5_all.snap",
        "allow_unauthenticated": true,
        "version": "1.0.5",
        "download_url": "https://public.apps.ubuntu.com/download/com.ubuntu.snappy/hello-world/hello-world_1.0.5_all.snap",
        "download_sha512": "3e8b192e18907d8195c2e380edd048870eda4f6dbcba8f65e4625d6efac3c37d11d607147568ade6f002b6baa30762c6da02e7ee462de7c56301ddbdc10d87f6"
    }
]
`

/* acquired via
   curl -H "accept: application/hal+json" -H "X-Ubuntu-Frameworks: ubuntu-core-15.04-dev1" https://search.apps.ubuntu.com/api/v1/package/com.ubuntu.snappy.xkcd-webserver
*/
const MockDetailsJSON = `
{
  "architecture": [
    "all"
  ],
  "allow_unauthenticated": true,
  "click_version": "0.1",
  "changelog": "",
  "date_published": "2014-12-05T13:12:31.785911Z",
  "license": "Apache License",
  "name": "xkcd-webserver",
  "publisher": "Canonical",
  "blacklist_country_codes": [],
  "icon_urls": {
    "256": "https:\/\/myapps.developer.ubuntu.com\/site_media\/appmedia\/2014\/12\/xkcd.svg.png"
  },
  "prices": null,
  "framework": [
    "ubuntu-core-15.04-dev1"
  ],
  "translations": null,
  "price": 0.0,
  "click_framework": [
    "ubuntu-core-15.04-dev1"
  ],
  "description": "Snappy\nThis is meant as a fun example for a snappy package.\r\n",
  "download_sha512": "3a9152b8bff494c036f40e2ca03d1dfaa4ddcfe651eae1c9419980596f48fa95b2f2a91589305af7d55dc08e9489b8392585bbe2286118550b288368e5d9a620",
  "website": "",
  "screenshot_urls": [],
  "department": [
    "entertainment"
  ],
  "company_name": "Canonical",
  "_links": {
    "self": {
      "href": "https:\/\/search.apps.ubuntu.com\/api\/v1\/package\/com.ubuntu.snappy.xkcd-webserver"
    },
    "curies": [
      {
        "templated": true,
        "name": "clickindex",
        "href": "https:\/\/search.apps.ubuntu.com\/docs\/v1\/relations.html{#rel}"
      }
    ]
  },
  "version": "0.3.1",
  "developer_name": "Snappy App Dev",
  "content": "application",
  "anon_download_url": "https:\/\/public.apps.ubuntu.com\/anon\/download\/com.ubuntu.snappy\/xkcd-webserver\/com.ubuntu.snappy.xkcd-webserver_0.3.1_all.click",
  "binary_filesize": 21236,
  "icon_url": "https:\/\/myapps.developer.ubuntu.com\/site_media\/appmedia\/2014\/12\/xkcd.svg.png",
  "support_url": "mailto:michael.vogt@ubuntu.com",
  "title": "Show random XKCD compic via a build-in webserver",
  "ratings_average": 0.0,
  "id": 1287,
  "screenshot_url": null,
  "terms_of_service": "",
  "download_url": "https:\/\/public.apps.ubuntu.com\/download\/com.ubuntu.snappy\/xkcd-webserver\/com.ubuntu.snappy.xkcd-webserver_0.3.1_all.click",
  "video_urls": [],
  "keywords": [
    "snappy"
  ],
  "video_embedded_html_urls": [],
  "last_updated": "2014-12-05T12:33:05.928364Z",
  "status": "Published",
  "whitelist_country_codes": []
}`
const MockNoDetailsJSON = `{"errors": ["No such package"], "result": "error"}`

type MockUbuntuStoreServer struct {
	quit chan int

	searchURI string
}

func (s *SnapTestSuite) TestUbuntuStoreRepositorySearch(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, MockSearchJSON)
	}))
	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	snap := newUbuntuStoreSnapRepository()
	c.Assert(snap, NotNil)
	snap.searchURI = mockServer.URL + "/%s"

	results, err := snap.Search("xkcd")
	c.Assert(err, IsNil)
	c.Assert(len(results), Equals, 1)
	c.Assert(results[0].Name(), Equals, "xkcd-webserver.mvo")
	c.Assert(results[0].Version(), Equals, "0.1")
	c.Assert(results[0].Description(), Equals, "Show random XKCD comic")
}

func mockInstalledSnapNamesByType(mockSnaps []string) (mockRestorer func()) {
	origFunc := InstalledSnapNamesByType
	InstalledSnapNamesByType = func(snapTs ...SnapType) (res []string, err error) {
		return mockSnaps, nil
	}
	return func() {
		InstalledSnapNamesByType = origFunc
	}
}

func (s *SnapTestSuite) TestUbuntuStoreRepositoryUpdates(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonReq, err := ioutil.ReadAll(r.Body)
		c.Assert(err, IsNil)
		c.Assert(string(jsonReq), Equals, `{"name":["hello-world"]}`)
		io.WriteString(w, MockUpdatesJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	snap := newUbuntuStoreSnapRepository()
	c.Assert(snap, NotNil)
	snap.bulkURI = mockServer.URL + "/updates/"

	// override the real InstalledSnapNamesByType to return our
	// mock data
	mockRestorer := mockInstalledSnapNamesByType([]string{"hello-world"})
	defer mockRestorer()

	// the actual test
	results, err := snap.Updates()
	c.Assert(err, IsNil)
	c.Assert(len(results), Equals, 1)
	c.Assert(results[0].Name(), Equals, "hello-world")
	c.Assert(results[0].Version(), Equals, "1.0.5")
}

func (s *SnapTestSuite) TestUbuntuStoreRepositoryUpdatesNoSnaps(c *C) {

	snap := newUbuntuStoreSnapRepository()
	c.Assert(snap, NotNil)

	// ensure we do not hit the net if there is nothing installed
	// (otherwise the store will send us all snaps)
	snap.bulkURI = "http://i-do.not-exist.really-not"
	mockRestorer := mockInstalledSnapNamesByType([]string{})
	defer mockRestorer()

	// the actual test
	results, err := snap.Updates()
	c.Assert(err, IsNil)
	c.Assert(len(results), Equals, 0)
}

func (s *SnapTestSuite) TestUbuntuStoreRepositoryDetails(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(strings.HasSuffix(r.URL.String(), "xkcd-webserver"), Equals, true)
		io.WriteString(w, MockDetailsJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	snap := newUbuntuStoreSnapRepository()
	c.Assert(snap, NotNil)
	snap.detailsURI = mockServer.URL + "/details/%s"

	// the actual test
	results, err := snap.Details("xkcd-webserver")
	c.Assert(err, IsNil)
	c.Assert(len(results), Equals, 1)
	c.Assert(results[0].Name(), Equals, "xkcd-webserver")
	c.Assert(results[0].Version(), Equals, "0.3.1")
}

func (s *SnapTestSuite) TestUbuntuStoreRepositoryNoDetails(c *C) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(strings.HasSuffix(r.URL.String(), "no-such-pkg"), Equals, true)
		w.WriteHeader(404)
		io.WriteString(w, MockNoDetailsJSON)
	}))

	c.Assert(mockServer, NotNil)
	defer mockServer.Close()

	snap := newUbuntuStoreSnapRepository()
	c.Assert(snap, NotNil)
	snap.detailsURI = mockServer.URL + "/details/%s"

	// the actual test
	results, err := snap.Details("no-such-pkg")
	c.Assert(len(results), Equals, 0)
	c.Assert(err, NotNil)
}
