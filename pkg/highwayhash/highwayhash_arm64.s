//+build !noasm !appengine

TEXT ·updateArm64(SB), 7, $0
	MOVD state+0(FP), R0
	MOVD msg_base+8(FP), R1
	MOVD msg_len+16(FP), R2
	SUBS $32, R2
	BMI  complete

	MOVD $·constants(SB), R3

	WORD $0x4c40607c

	WORD $0x4cdf2c00
	WORD $0x4c402c04
	SUBS $64, R0

loop:
	WORD $0x4cdfa83a

	WORD $0x4efa8442
	WORD $0x4efb8463

	WORD $0x4ee48442
	WORD $0x4ee58463

	WORD $0x4e1d200a
	WORD $0x4e1e204b
	WORD $0x2eaac16c
	WORD $0x6eaac16d

	WORD $0x4ee68400
	WORD $0x4ee78421

	WORD $0x4e1d204f
	WORD $0x4e1e200e

	WORD $0x6e2c1c84
	WORD $0x6e2d1ca5

	WORD $0x2eaec1f0
	WORD $0x6eaec1f1

	WORD $0x4e1c0052
	WORD $0x4ef28400
	WORD $0x4e1c0073
	WORD $0x4ef38421

	WORD $0x4e1c0014
	WORD $0x4ef48442
	WORD $0x4e1c0035
	WORD $0x4ef58463

	WORD $0x6e301cc6
	WORD $0x6e311ce7

	SUBS $32, R2
	BPL  loop

	WORD $0x4c9f2c00
	WORD $0x4c002c04

complete:
	RET

DATA ·constants+0x0(SB)/8, $0x000f010e05020c03
DATA ·constants+0x8(SB)/8, $0x070806090d0a040b
DATA ·constants+0x10(SB)/8, $0x0f0e0d0c07060504
DATA ·constants+0x18(SB)/8, $0x1f1e1d1c17161514
DATA ·constants+0x20(SB)/8, $0x0b0a090803020100
DATA ·constants+0x28(SB)/8, $0x1b1a191813121110

GLOBL ·constants(SB), 8, $48
