/*
 * Copyright (C) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package snappy

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "launchpad.net/gocheck"
)

const configPassthroughScript = `#!/bin/sh

# temp location to store cfg
CFG="%s/config.out"

# just dump out for the tes
cat - > $CFG

# and config
cat $CFG
`

const configErrorScript = `#!/bin/sh

printf "error: some error"
exit 1
`

const mockAaExecScript = `#!/bin/sh
[ "$1" = "-d" ]
shift
[ -n "$1" ]
shift
exec "$@"
`

const configYaml = `
config:
  hello-world:
    key: value
`

func (s *SnapTestSuite) makeInstalledMockSnapWithConfig(c *C, configScript string) (snapDir string, err error) {
	yamlFile, err := s.makeInstalledMockSnap()
	c.Assert(err, IsNil)
	metaDir := filepath.Dir(yamlFile)
	err = os.Mkdir(filepath.Join(metaDir, "hooks"), 0755)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(filepath.Join(metaDir, "hooks", "config"), []byte(configScript), 0755)
	c.Assert(err, IsNil)

	snapDir = filepath.Dir(metaDir)
	return snapDir, nil
}

func (s *SnapTestSuite) TestConfigSimple(c *C) {
	mockConfig := fmt.Sprintf(configPassthroughScript, s.tempdir)
	snapDir, err := s.makeInstalledMockSnapWithConfig(c, mockConfig)
	c.Assert(err, IsNil)

	newConfig, err := snapConfig(snapDir, configYaml)
	c.Assert(err, IsNil)
	content, err := ioutil.ReadFile(filepath.Join(s.tempdir, "config.out"))
	c.Assert(err, IsNil)
	c.Assert(content, DeepEquals, []byte(configYaml))
	c.Assert(newConfig, Equals, configYaml)
}

func (s *SnapTestSuite) TestConfigError(c *C) {
	snapDir, err := s.makeInstalledMockSnapWithConfig(c, configErrorScript)
	c.Assert(err, IsNil)

	newConfig, err := snapConfig(snapDir, configYaml)
	c.Assert(err, NotNil)
	c.Assert(newConfig, Equals, "")
	c.Assert(strings.HasSuffix(err.Error(), "failed with: 'error: some error'"), Equals, true)
}
