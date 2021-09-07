package sha256

func cpuid(op uint32) (eax, ebx, ecx, edx uint32) {
	return 0, 0, 0, 0
}

func cpuidex(op, op2 uint32) (eax, ebx, ecx, edx uint32) {
	return 0, 0, 0, 0
}

func xgetbv(index uint32) (eax, edx uint32) {
	return 0, 0
}

func haveArmSha() bool {
	return false
}
