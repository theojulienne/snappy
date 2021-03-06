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

package main

import (
	"fmt"
	"os"

	"launchpad.net/snappy/priv"
	"launchpad.net/snappy/snappy"
)

type cmdUpdate struct {
}

func init() {
	var cmdUpdateData cmdUpdate
	_, _ = parser.AddCommand("update",
		"Update all installed parts",
		"Ensures system is running with latest parts",
		&cmdUpdateData)
}

func (x *cmdUpdate) Execute(args []string) (err error) {
	privMutex := priv.New()
	if err := privMutex.TryLock(); err != nil {
		return err
	}
	defer privMutex.Unlock()

	return update()
}

func update() error {
	// FIXME: handle args
	updates, err := snappy.ListUpdates()
	if err != nil {
		return err
	}

	for _, part := range updates {
		pbar := snappy.NewTextProgress(part.Name())

		fmt.Printf("Installing %s (%s)\n", part.Name(), part.Version())
		if err := part.Install(pbar, 0); err != nil {
			return err
		}
	}

	if len(updates) > 0 {
		showVerboseList(updates, os.Stdout)
	}

	return nil
}
