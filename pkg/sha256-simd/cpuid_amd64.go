package sha256

func cpuid(op uint32) (eax, ebx, ecx, edx uint32)
func cpuidex(op, op2 uint32) (eax, ebx, ecx, edx uint32)
func xgetbv(index uint32) (eax, edx uint32)

func haveArmSha() bool {
	return false
}
