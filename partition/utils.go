package partition

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"bufio"
)

// Return true if given path exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return (err == nil)
}

// Return true if the given path exists and is a directory
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

// FIXME: would it make sense to differenciate between launch errors and
//        exit code? (i.e. something like (returnCode, error) ?)
func runCommandImpl(args ...string) (err error) {
	if len(args) == 0 {
		return errors.New("ERROR: no command specified")
	}

	// FIXME: use logger
	/*
		if debug == true {

			log.debug('running: {}'.format(args))
		}
	*/

	if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil {
		cmdline := strings.Join(args, " ")
		return errors.New(fmt.Sprintf("Failed to run command '%s': %s (%s)",
			cmdline,
			out,
			err))
	}
	return nil
}

// Run the command specified by args
// This is a var instead of a function to making mocking in the tests easier
var runCommand = runCommandImpl

// Run command specified by args and return array of output lines.
// FIXME: would it make sense to make this a vararg (args...) ?
func runCommandWithStdout(args ...string) (output []string, err error) {
	if len(args) == 0 {
		return []string{}, errors.New("ERROR: no command specified")
	}

	// FIXME: use logger
	/*
		if debug == true {

			log.debug('running: {}'.format(args))
		}
	*/

	bytes, err := exec.Command(args[0], args[1:]...).Output()
	if err != nil {
		return output, err
	}

	output = strings.Split(string(bytes), "\n")

	// remove last element if it's empty
	if len(output) > 1 {
		last := output[len(output)-1]
		if last == "" {
			output = output[:len(output)-1]
		}
	}

	return output, err
}

// Return a string slice of all lines in file specified by path 
func readLines(path string) (lines []string, err error) {

	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// Write lines slice to file specified by path
func writeLines(lines []string, path string) (err error) {

	file, err := os.Create(path)

	if err != nil {
		return err
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, line := range lines {
		if _, err = fmt.Fprintln(writer, line); err != nil {
			return err
		}
	}
	return writer.Flush()
}

// Write lines to file atomically. File does not have to preexist.
func atomicFileUpdate(file string, lines []string) (err error) {
	tmpFile := fmt.Sprintf("%s.NEW", file)

	if err := writeLines(lines, tmpFile); err != nil {
		return err
	}

	// atomic update
	if err = os.Rename(tmpFile, file); err != nil {
		return err
	}

	return err
}

// Rewrite the specified file, applying the specified set of changes.
// Lines not in the changes slice are left alone.
// If the original file does not contain any of the name entries (from
// the corresponding ConfigFileChange objects), those entries are
// appended to the file.
//
func modifyNameValueFile(file string, changes []ConfigFileChange) (err error) {
	var lines []string
	var updated []ConfigFileChange

	if lines, err = readLines(file); err != nil {
		return err
	}

	var new []string

	for _, line := range lines {
		for _, change := range changes {
			if strings.HasPrefix(line, fmt.Sprintf("%s=", change.Name)) {
				line = fmt.Sprintf("%s=%s", change.Name, change.Value)
				updated = append(updated, change)
			}
		}
		new = append(new, line)
	}

	lines = new

	for _, change := range changes {
		var got bool = false
		for _, update := range updated {
			if update.Name == change.Name {
				got = true
				break
			}
		}

		if got == false {
			// name/value pair did not exist in original
			// file, so append
			lines = append(lines, fmt.Sprintf("%s=%s",
				change.Name, change.Value))
		}
	}

	return atomicFileUpdate(file, lines)
}

func getLsblkOutput() ([]string, error) {
	return runCommandWithStdout(
		"/bin/lsblk",
		"--ascii",
		"--output=NAME,LABEL,PKNAME,MOUNTPOINT",
		"--pairs")
}

