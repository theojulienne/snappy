package snappy

import (
	"io/ioutil"
	"os"

	. "launchpad.net/gocheck"
)

func makeCloudInitMetaData(c *C, content string) string {
	w, err := ioutil.TempFile("", "meta-data")
	c.Assert(err, IsNil)
	w.Write([]byte(content))
	w.Sync()
	return w.Name()
}

func (s *SnapTestSuite) TestNotInDeveloperMode(c *C) {
	cloudMetaDataFile = makeCloudInitMetaData(c, `instance-id: nocloud-static`)
	defer os.Remove(cloudMetaDataFile)
	c.Assert(inDeveloperMode(), Equals, false)
}

func (s *SnapTestSuite) TestInDeveloperMode(c *C) {
	cloudMetaDataFile = makeCloudInitMetaData(c, `instance-id: nocloud-static
public-keys:
  - ssh-rsa AAAAB3NzAndSoOn
`)
	defer os.Remove(cloudMetaDataFile)
	c.Assert(inDeveloperMode(), Equals, true)
}

func (s *SnapTestSuite) TestInstallOemFails(c *C) {
	snapFile := makeTestSnapPackage(c, `name: foo
version: 1.0
type: oem
icon: foo.svg
vendor: Foo Bar <foo@example.com>`)
	os.Setenv("SNAPPY_ALLOW_OEM_INSTALL", "")

	err := Install([]string{snapFile})
	c.Assert(err, Equals, ErrPackageNotInstallable)
}

func (s *SnapTestSuite) TestInstallOemWorks(c *C) {
	snapFile := makeTestSnapPackage(c, `name: foo
version: 1.0
type: oem
icon: foo.svg
vendor: Foo Bar <foo@example.com>`)
	os.Setenv("SNAPPY_ALLOW_OEM_INSTALL", "1")

	err := Install([]string{snapFile})
	c.Assert(err, Equals, nil)
}
