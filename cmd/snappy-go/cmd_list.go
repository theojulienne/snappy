package main

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"launchpad.net/snappy/snappy"
)

type CmdList struct {
	Updates bool `short:"u" long:"updates" description:"Show available updates"`
	ShowAll bool `short:"a" long:"all" description:"Show all parts"`
}

var cmdList CmdList

func init() {
	cmd, _ := Parser.AddCommand("list",
		"List installed parts",
		"Shows all installed parts",
		&cmdList)

	cmd.Aliases = append(cmd.Aliases, "li")
}

func (x *CmdList) Execute(args []string) (err error) {
	return x.list()
}

func (x CmdList) list() error {
	installed, err := snappy.ListInstalled()
	if err != nil {
		return err
	}

	if x.Updates {
		updates, err := snappy.ListUpdates()
		if err != nil {
			return err
		}
		showUpdatesList(installed, updates, x.ShowAll, os.Stdout)
	} else {
		showInstalledList(installed, x.ShowAll, os.Stdout)
	}

	return err
}

func showRebootMessage(installed []snappy.Part, o io.Writer) {
	// Initialise to handle systems without a provisioned "other"
	otherVersion := "0"
	currentVersion := "0"
	otherName := ""
	needsReboot := false

	for _, part := range installed {
		// FIXME: extend this later to look at more than just
		//        core - once we do that the logic here needs
		//        to be modified as the current code assumes
		//        there are only two version instaleld and
		//        there is only a single part that may requires
		//        a reboot
		if part.Type() != snappy.SnapTypeCore {
			continue
		}

		if part.NeedsReboot() {
			needsReboot = true
		}

		if part.IsActive() {
			currentVersion = part.Version()
		} else {
			otherVersion = part.Version()
			otherName = part.Name()
		}
	}

	if needsReboot {
		if snappy.VersionCompare(otherVersion, currentVersion) > 0 {
			fmt.Fprintln(o, fmt.Sprintf("Reboot to use the new %s.", otherName))
		} else {
			fmt.Fprintln(o, fmt.Sprintf("Reboot to use %s version %s.", otherName, otherVersion))
		}
	}
}

func showInstalledList(installed []snappy.Part, showAll bool, o io.Writer) {
	w := tabwriter.NewWriter(o, 5, 3, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Name\tVersion\tSummary\t")
	for _, part := range installed {
		if showAll || part.IsActive() {
			fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t", part.Name(), part.Version(), part.Description()))
		}
	}
	showRebootMessage(installed, o)
}

func showUpdatesList(installed []snappy.Part, updates []snappy.Part, showAll bool, o io.Writer) {
	w := tabwriter.NewWriter(o, 5, 3, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Name\tVersion\tUpdate\t")
	for _, part := range installed {
		if showAll || part.IsActive() {
			update := snappy.FindPartByName(part.Name(), updates)
			ver := "-"
			if update != nil {
				ver = (*update).Version()
			}
			fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t", part.Name(), part.Version(), ver))
		}
	}
}
