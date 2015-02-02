package main

import "launchpad.net/snappy/snappy"

type CmdCacheLsblk struct {
}

var cmdCacheLsblk CmdCacheLsblk

func init() {
	Parser.AddCommand("cache-lsblk",
		"Cache lsblk(8) data (INTERNAL)",
		"Do not run this command manually",
		&cmdCacheLsblk)
}

func (x *CmdCacheLsblk) Execute(args []string) (err error) {
	parts, err := snappy.InstalledSnapsByType(snappy.SnapTypeCore)
	if err != nil {
		return err
	}

	return parts[0].(*snappy.SystemImagePart).GenerateLsblkCache()
}
