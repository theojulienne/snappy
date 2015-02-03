//--------------------------------------------------------------------
// Copyright (c) 2014-2015 Canonical Ltd.
//--------------------------------------------------------------------

package partition

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	BOOTLOADER_UBOOT_DIR         = "/boot/uboot"
	BOOTLOADER_UBOOT_CONFIG_FILE = "/boot/uboot/uEnv.txt"

	// File created by u-boot itself when
	// BOOTLOADER_BOOTMODE_VAR_START_VALUE == "try" which the
	// successfully booted system must remove to flag to u-boot that
	// this partition is "good".
	BOOTLOADER_UBOOT_STAMP_FILE = "/boot/uboot/snappy-stamp.txt"

	// the main uEnv.txt u-boot config file sources this snappy
	// boot-specific config file.
	BOOTLOADER_UBOOT_ENV_FILE = "snappy-system.txt"
)

type Uboot struct {
	*BootLoaderType
}

// Stores a Name and a Value to be added as a name=value pair in a file.
type ConfigFileChange struct {
	Name  string
	Value string
}

// Create a new Grub bootloader object
func NewUboot(partition *Partition) *Uboot {
	u := Uboot{BootLoaderType: NewBootLoader(partition)}

	u.currentBootPath = path.Join(BOOTLOADER_UBOOT_DIR, u.currentRootfs)

	u.otherBootPath = path.Join(BOOTLOADER_UBOOT_DIR, u.otherRootfs)

	return &u
}

func (u *Uboot) Name() string {
	// XXX: same value as used in HARDWARE_SPEC_FILE
	return "u-boot"
}

func (u *Uboot) Installed() bool {
	// crude heuristic
	return fileExists(BOOTLOADER_UBOOT_CONFIG_FILE)
}

// Make the U-Boot bootloader switch rootfs's.
//
// Approach:
//
// - Assume the device's installed version of u-boot supports
//   CONFIG_SUPPORT_RAW_INITRD (that allows u-boot to boot a
//   standard initrd+kernel on the fat32 disk partition).
// - Copy the "other" rootfs's kernel+initrd to the boot partition,
//   renaming them in the process to ensure the next boot uses the
//   correct versions.
func (u *Uboot) ToggleRootFS() (err error) {

	// write 1 character partition name ('a' or 'b')
	value := string(u.otherLabel[len(u.otherLabel)-1])

	// If the file exists, update it. Otherwise create it.
	//
	// The file _should_ always exist, but since it's on a writable
	// partition, it's possible the admin removed it by mistake. So
	// recreate to allow the system to boot!
	changes := []ConfigFileChange{
		ConfigFileChange{Name: BOOTLOADER_ROOTFS_VAR,
			Value: value,
		},
		ConfigFileChange{Name: BOOTLOADER_BOOTMODE_VAR,
			Value: BOOTLOADER_BOOTMODE_VAR_START_VALUE,
		},
	}

	return modifyNameValueFile(BOOTLOADER_UBOOT_ENV_FILE, changes)
}

func (u *Uboot) GetAllBootVars() (vars []string, err error) {
	return getNameValuePairs(BOOTLOADER_UBOOT_ENV_FILE)
}

func (u *Uboot) GetBootVar(name string) (value string, err error) {
	var vars []string

	vars, err = u.GetAllBootVars()

	if err != nil {
		return value, err
	}

	for _, pair := range vars {
		fields := strings.Split(string(pair), "=")

		if fields[0] == name {
			return fields[1], err
		}
	}

	return value, err
}

func (u *Uboot) SetBootVar(name, value string) (err error) {
	var lines []string

	if lines, err = readLines(BOOTLOADER_UBOOT_ENV_FILE); err != nil {
		return err
	}

	new := fmt.Sprintf("%s=%s", name, value)
	lines = append(lines, new)

	// Rewrite the file
	return atomicFileUpdate(BOOTLOADER_UBOOT_ENV_FILE, lines)
}

func (u *Uboot) ClearBootVar(name string) (currentValue string, err error) {
	var saved []string
	var lines []string

	// XXX: note that we do not call GetAllBootVars() since that
	// strips all comments (which we want to retain).
	if lines, err = readLines(BOOTLOADER_UBOOT_ENV_FILE); err != nil {
		return currentValue, err
	}

	for _, line := range lines {
		fields := strings.Split(string(line), "=")
		if fields[0] == name {
			currentValue = fields[1]
		} else {
			saved = append(saved, line)
		}
	}

	// Rewrite the file, excluding the name to clear
	return currentValue, atomicFileUpdate(BOOTLOADER_UBOOT_ENV_FILE, saved)
}

func (u *Uboot) GetNextBootRootFSName() (label string, err error) {
	var value string

	if value, err = u.GetBootVar(BOOTLOADER_ROOTFS_VAR); err != nil {
		// should never happen
		return label, err
	}

	return value, err
}

func (u *Uboot) GetRootFSName() string {
	return u.currentRootfs
}

func (u *Uboot) GetOtherRootFSName() string {
	return u.otherRootfs
}

// Returns name=value entries from the specified file, removing all
// blank lines and comments.
func getNameValuePairs(file string) (vars []string, err error) {
	var lines []string

	if lines, err = readLines(file); err != nil {
		return vars, err
	}

	for _, line := range lines {
		// ignore blank lines
		if line == "" || line == "\n" {
			continue
		}

		// ignore comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Index(line, "=") != -1 {
			vars = append(vars, line)
		}
	}

	return vars, err
}

func (u *Uboot) MarkCurrentBootSuccessful() (err error) {
	changes := []ConfigFileChange{
		ConfigFileChange{Name: BOOTLOADER_BOOTMODE_VAR,
			Value: BOOTLOADER_BOOTMODE_VAR_END_VALUE,
		},
	}

	err = modifyNameValueFile(BOOTLOADER_UBOOT_ENV_FILE, changes)
	if err != nil {
		return err
	}

	return os.RemoveAll(BOOTLOADER_UBOOT_STAMP_FILE)
}

func (u *Uboot) SyncBootFiles() (err error) {
	srcDir := u.currentBootPath
	destDir := u.otherBootPath

	// always start from scratch: all files here are owned by us.
	os.RemoveAll(destDir)

	return runCommand("/bin/cp", "-a", srcDir, destDir)
}

func (u *Uboot) HandleAssets() (err error) {

	var dirsToRemove map[string]int

	dirsToRemove = make(map[string]int)

	defer func() {
		var dirs []string

		// convert to slice
		for dir, _ := range dirsToRemove {
			dirs = append(dirs, dir)
		}

		// reverse sort to ensure a depth-first approach
		sort.Sort(sort.Reverse(sort.StringSlice(dirs)))

		for _, dir := range dirs {
			if err = os.RemoveAll(dir); err != nil {
				panic(err)
			}
		}
	}()

	hardware, err := u.partition.hardwareSpec()
	if err != nil {
		return err
	}

	// validate
	if hardware.Bootloader != u.Name() {
		panic(fmt.Sprintf(
			"ERROR: bootloader is of type %s but hardware spec requires %s",
			u.Name(),
			hardware.Bootloader))
	}

	// validate
	switch hardware.PartitionLayout {
	case "system-AB":
		if u.partition.dualRootPartitions() != true {
			panic(fmt.Sprintf(
				"ERROR: hardware spec requires dual root partitions"))
		}
	}

	destDir := u.otherBootPath

	err = os.MkdirAll(destDir, DIR_MODE)
	if err != nil {
		return err
	}

	// install kernel+initrd
	for _, file := range []string{hardware.Kernel, hardware.Initrd} {

		if file == "" {
			continue
		}

		// expand path
		path := path.Join(u.partition.cacheDir(), file)

		if !fileExists(path) {
			continue
		}

		dir := filepath.Dir(path)
		dirsToRemove[dir] = 1

		err = runCommand("/bin/cp", file, destDir)
		if err != nil {
			return err
		}
	}

	// install .dtb files
	if fileExists(hardware.DtbDir) {
		dtbDestDir := path.Join(destDir, "dtbs")

		err = os.MkdirAll(dtbDestDir, DIR_MODE)
		if err != nil {
			return err
		}

		files, err := filepath.Glob(path.Join(hardware.DtbDir, "*"))
		if err != nil {
			return err
		}

		for _, file := range files {
			err = runCommand("/bin/cp", file, dtbDestDir)
			if err != nil {
				return err
			}
		}
	}

	flashAssetsDir := u.partition.flashAssetsDir()

	if fileExists(flashAssetsDir) {
		// FIXME: we don't currently do anything with the
		// MLO + uImage files since they are not specified in
		// the hardware spec. So for now, just remove them.

		err = os.RemoveAll(flashAssetsDir)
		if err != nil {
			return err
		}
	}

	return err
}

