package snappy

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func cmdBuild(args []string) error {

	dir := args[0]

	// FIXME this functions suxx, its just proof-of-concept

	data, err := ioutil.ReadFile(dir + "/meta/package.yaml")
	if err != nil {
		return err
	}
	m, err := getMapFromYaml(data)
	if err != nil {
		return err
	}

	arch := m["architecture"]
	if arch == nil {
		arch = "all"
	} else {
		arch = arch.(string)
	}
	output_name := fmt.Sprintf("%s_%s_%s.snap", m["name"], m["version"], arch)

	os.Chdir(dir)
	cmd := exec.Command("tar", "czf", "meta.tar.gz", "meta/")
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	cmd = exec.Command("mksquashfs", ".", "data.squashfs", "-comp", "xz")
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	os.Remove(output_name)
	cmd = exec.Command("ar", "q", output_name, "meta.tar.gz", "data.squashfs")
	err = cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	os.Remove("meta.tar.gz")
	os.Remove("data.squashfs")

	return nil
}