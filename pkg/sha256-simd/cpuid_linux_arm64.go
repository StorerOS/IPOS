// +build arm64,linux

package sha256

import (
	"bytes"
	"io/ioutil"
)

func cpuid(op uint32) (eax, ebx, ecx, edx uint32) {
	return 0, 0, 0, 0
}

func cpuidex(op, op2 uint32) (eax, ebx, ecx, edx uint32) {
	return 0, 0, 0, 0
}

func xgetbv(index uint32) (eax, edx uint32) {
	return 0, 0
}

const procCPUInfo = "/proc/cpuinfo"

const sha256Feature = "sha2"

func haveArmSha() bool {
	cpuInfo, err := ioutil.ReadFile(procCPUInfo)
	if err != nil {
		return false
	}
	return bytes.Contains(cpuInfo, []byte(sha256Feature))
}
