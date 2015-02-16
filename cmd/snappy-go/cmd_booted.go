package main

import "launchpad.net/snappy/snappy"

type cmdBooted struct {
}

func init() {
	var cmdBootedData cmdBooted
	parser.AddCommand("booted",
		"Flag that rootfs booted successfully",
		"Not necessary to run this command manually",
		&cmdBootedData)
}

func (x *cmdBooted) Execute(args []string) (err error) {
	if !isRoot() {
		return ErrRequiresRoot
	}

	return snappy.MarkBootSuccessful()
}
