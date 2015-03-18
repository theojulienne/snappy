package partition

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// Representation of HARDWARE_SPEC_FILE
type hardwareSpecType struct {
	Kernel          string         `yaml:"kernel"`
	Initrd          string         `yaml:"initrd"`
	DtbDir          string         `yaml:"dtbs"`
	PartitionLayout string         `yaml:"partition-layout"`
	Bootloader      bootloaderName `yaml:"bootloader"`

	hardware struct {
		DtbFile string `yaml:"dtb"`
	} `yaml:"hardware"`
}

func parseHardwareYaml(hardwareSpecFile string) (hardwareSpecType, error) {
	h := hardwareSpecType{}

	data, err := ioutil.ReadFile(hardwareSpecFile)
	// if hardware.yaml does not exist it just means that there was no
	// device part in the update.
	if os.IsNotExist(err) {
		return h, ErrNoHardwareYaml
	} else if err != nil {
		return h, err
	}

	if err := yaml.Unmarshal([]byte(data), &h); err != nil {
		return h, err
	}

	return h, nil
}
