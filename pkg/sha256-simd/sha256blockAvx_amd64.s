//+build !noasm,!appengine

#include "textflag.h"

#define ROTATE_XS \
	MOVOU X4, X15 \
	MOVOU X5, X4  \
	MOVOU X6, X5  \
	MOVOU X7, X6  \
	MOVOU X15, X7

#define FOUR_ROUNDS_AND_SCHED(a, b, c, d, e, f, g, h) \
	MOVL e, R13                    \
	ROLL $18, R13                  \
	MOVL a, R14                    \
	LONG $0x0f41e3c4; WORD $0x04c6 \
	ROLL $23, R14                  \
	XORL e, R13                    \
	MOVL f, R15                    \
	ROLL $27, R13                  \
	XORL a, R14                    \
	XORL g, R15                    \
	LONG $0xc4fef9c5               \
	XORL e, R13                    \
	ANDL e, R15                    \
	ROLL $21, R14                  \
	                               \
	\
	                               \
	LONG $0x0f51e3c4; WORD $0x04cc \
	XORL a, R14                    \
	ROLL $26, R13                  \
	XORL g, R15                    \
	ROLL $30, R14                  \
	ADDL R13, R15                  \
	ADDL _xfer+48(FP), R15         \
	MOVL a, R13                    \
	ADDL R15, h                    \
	\
	MOVL a, R15                    \
	LONG $0xd172e9c5; BYTE $0x07   \
	ORL  c, R13                    \
	ADDL h, d                      \
	ANDL c, R15                    \
	LONG $0xf172e1c5; BYTE $0x19   \
	ANDL b, R13                    \
	ADDL R14, h                    \
	LONG $0xdaebe1c5               \
	ORL  R15, R13                  \
	ADDL R13, h                    \
	\
	MOVL d, R13                    \
	MOVL h, R14                    \
	ROLL $18, R13                  \
	XORL d, R13                    \
	MOVL e, R15                    \
	ROLL $23, R14                  \
	LONG $0xd172e9c5; BYTE $0x12   \
	XORL h, R14                    \
	ROLL $27, R13                  \
	XORL f, R15                    \
	LONG $0xd172b9c5; BYTE $0x03   \
	ROLL $21, R14                  \
	XORL d, R13                    \
	ANDL d, R15                    \
	ROLL $26, R13                  \
	LONG $0xf172f1c5; BYTE $0x0e   \
	XORL h, R14                    \
	XORL f, R15                    \
	LONG $0xd9efe1c5               \
	ADDL R13, R15                  \
	ADDL _xfer+52(FP), R15         \
	ROLL $30, R14                  \
	LONG $0xdaefe1c5               \
	MOVL h, R13                    \
	ADDL R15, g                    \
	MOVL h, R15                    \
	LONG $0xef61c1c4; BYTE $0xc8   \
	ORL  b, R13                    \
	ADDL g, c                      \
	ANDL b, R15                    \
	                               \
	\
	                               \
	LONG $0xd770f9c5; BYTE $0xfa   \
	ANDL a, R13                    \
	ADDL R14, g                    \
	LONG $0xc1fef9c5               \
	ORL  R15, R13                  \
	ADDL R13, g                    \
	\
	MOVL c, R13                    \
	MOVL g, R14                    \
	ROLL $18, R13                  \
	XORL c, R13                    \
	ROLL $23, R14                  \
	MOVL d, R15                    \
	XORL g, R14                    \
	ROLL $27, R13                  \
	LONG $0xd272b9c5; BYTE $0x0a   \
	XORL e, R15                    \
	LONG $0xd273e1c5; BYTE $0x13   \
	XORL c, R13                    \
	ANDL c, R15                    \
	LONG $0xd273e9c5; BYTE $0x11   \
	ROLL $21, R14                  \
	XORL g, R14                    \
	XORL e, R15                    \
	ROLL $26, R13                  \
	LONG $0xd3efe9c5               \
	ADDL R13, R15                  \
	ROLL $30, R14                  \
	ADDL _xfer+56(FP), R15         \
	LONG $0xc2ef39c5               \
	MOVL g, R13                    \
	ADDL R15, f                    \
	MOVL g, R15                    \
	LONG $0x003942c4; BYTE $0xc2   \
	ORL  a, R13                    \
	ADDL f, b                      \
	ANDL a, R15                    \
	LONG $0xfe79c1c4; BYTE $0xc0   \
	ANDL h, R13                    \
	ADDL R14, f                    \
	                               \
	\
	                               \
	LONG $0xd070f9c5; BYTE $0x50   \
	ORL  R15, R13                  \
	ADDL R13, f                    \
	\
	MOVL b, R13                    \
	ROLL $18, R13                  \
	MOVL f, R14                    \
	ROLL $23, R14                  \
	XORL b, R13                    \
	MOVL c, R15                    \
	ROLL $27, R13                  \
	LONG $0xd272a1c5; BYTE $0x0a   \
	XORL f, R14                    \
	XORL d, R15                    \
	LONG $0xd273e1c5; BYTE $0x13   \
	XORL b, R13                    \
	ANDL b, R15                    \
	ROLL $21, R14                  \
	LONG $0xd273e9c5; BYTE $0x11   \
	XORL f, R14                    \
	ROLL $26, R13                  \
	XORL d, R15                    \
	LONG $0xd3efe9c5               \
	ROLL $30, R14                  \
	ADDL R13, R15                  \
	ADDL _xfer+60(FP), R15         \
	LONG $0xdaef21c5               \
	MOVL f, R13                    \
	ADDL R15, e                    \
	MOVL f, R15                    \
	LONG $0x002142c4; BYTE $0xdc   \
	ORL  h, R13                    \
	ADDL e, a                      \
	ANDL h, R15                    \
	LONG $0xe0fea1c5               \
	ANDL g, R13                    \
	ADDL R14, e                    \
	ORL  R15, R13                  \
	ADDL R13, e                    \
	\
	ROTATE_XS

#define DO_ROUND(a, b, c, d, e, f, g, h, offset) \
	MOVL e, R13                \
	ROLL $18, R13              \
	MOVL a, R14                \
	XORL e, R13                \
	ROLL $23, R14              \
	MOVL f, R15                \
	XORL a, R14                \
	ROLL $27, R13              \
	XORL g, R15                \
	XORL e, R13                \
	ROLL $21, R14              \
	ANDL e, R15                \
	XORL a, R14                \
	ROLL $26, R13              \
	XORL g, R15                \
	ADDL R13, R15              \
	ROLL $30, R14              \
	ADDL _xfer+offset(FP), R15 \
	MOVL a, R13                \
	ADDL R15, h                \
	MOVL a, R15                \
	ORL  c, R13                \
	ADDL h, d                  \
	ANDL c, R15                \
	ANDL b, R13                \
	ADDL R14, h                \
	ORL  R15, R13              \
	ADDL R13, h               

TEXT ·blockAvx(SB), 7, $0-80

	MOVQ h+0(FP), SI            
	MOVQ message_base+24(FP), R8
	MOVQ message_len+32(FP), R9 
	CMPQ R9, $0
	JEQ  done_hash
	ADDQ R8, R9
	MOVQ R9, reserved2+64(FP)   

	MOVL (0*4)(SI), AX 
	MOVL (1*4)(SI), BX 
	MOVL (2*4)(SI), CX 
	MOVL (3*4)(SI), R8 
	MOVL (4*4)(SI), DX 
	MOVL (5*4)(SI), R9 
	MOVL (6*4)(SI), R10
	MOVL (7*4)(SI), R11

	MOVOU bflipMask<>(SB), X13
	MOVOU shuf00BA<>(SB), X10 
	MOVOU shufDC00<>(SB), X12 

	MOVQ message_base+24(FP), SI

loop0:
	LEAQ constants<>(SB), BP


	MOVOU 0*16(SI), X4
	LONG  $0x0059c2c4; BYTE $0xe5
	MOVOU 1*16(SI), X5
	LONG  $0x0051c2c4; BYTE $0xed
	MOVOU 2*16(SI), X6
	LONG  $0x0049c2c4; BYTE $0xf5
	MOVOU 3*16(SI), X7
	LONG  $0x0041c2c4; BYTE $0xfd

	MOVQ SI, reserved3+72(FP)
	MOVD $0x3, DI


loop1:
	LONG  $0x4dfe59c5; BYTE $0x00
	MOVOU X9, reserved0+48(FP)
	FOUR_ROUNDS_AND_SCHED(AX, BX,  CX,  R8, DX, R9, R10, R11)

	LONG  $0x4dfe59c5; BYTE $0x10
	MOVOU X9, reserved0+48(FP)
	FOUR_ROUNDS_AND_SCHED(DX, R9, R10, R11, AX, BX,  CX,  R8)

	LONG  $0x4dfe59c5; BYTE $0x20
	MOVOU X9, reserved0+48(FP)
	FOUR_ROUNDS_AND_SCHED(AX, BX,  CX,  R8, DX, R9, R10, R11)

	LONG  $0x4dfe59c5; BYTE $0x30
	MOVOU X9, reserved0+48(FP)
	ADDQ  $64, BP
	FOUR_ROUNDS_AND_SCHED(DX, R9, R10, R11, AX, BX,  CX,  R8)

	SUBQ $1, DI
	JNE  loop1

	MOVD $0x2, DI

loop2:
	LONG  $0x4dfe59c5; BYTE $0x00
	MOVOU X9, reserved0+48(FP)
	DO_ROUND( AX,  BX,  CX,  R8,  DX,  R9, R10, R11, 48)
	DO_ROUND(R11,  AX,  BX,  CX,  R8,  DX,  R9, R10, 52)
	DO_ROUND(R10, R11,  AX,  BX,  CX,  R8,  DX,  R9, 56)
	DO_ROUND( R9, R10, R11,  AX,  BX,  CX,  R8,  DX, 60)

	LONG  $0x4dfe51c5; BYTE $0x10
	MOVOU X9, reserved0+48(FP)
	ADDQ  $32, BP
	DO_ROUND( DX,  R9, R10, R11,  AX,  BX,  CX,  R8, 48)
	DO_ROUND( R8,  DX,  R9, R10, R11,  AX,  BX,  CX, 52)
	DO_ROUND( CX,  R8,  DX,  R9, R10, R11,  AX,  BX, 56)
	DO_ROUND( BX,  CX,  R8,  DX,  R9, R10, R11,  AX, 60)

	MOVOU X6, X4
	MOVOU X7, X5

	SUBQ $1, DI
	JNE  loop2

	MOVQ h+0(FP), SI   
	ADDL (0*4)(SI), AX 
	MOVL AX, (0*4)(SI)
	ADDL (1*4)(SI), BX 
	MOVL BX, (1*4)(SI)
	ADDL (2*4)(SI), CX 
	MOVL CX, (2*4)(SI)
	ADDL (3*4)(SI), R8 
	MOVL R8, (3*4)(SI)
	ADDL (4*4)(SI), DX 
	MOVL DX, (4*4)(SI)
	ADDL (5*4)(SI), R9 
	MOVL R9, (5*4)(SI)
	ADDL (6*4)(SI), R10
	MOVL R10, (6*4)(SI)
	ADDL (7*4)(SI), R11
	MOVL R11, (7*4)(SI)

	MOVQ reserved3+72(FP), SI
	ADDQ $64, SI
	CMPQ reserved2+64(FP), SI
	JNE  loop0

done_hash:
	RET

DATA constants<>+0x0(SB)/8, $0x71374491428a2f98
DATA constants<>+0x8(SB)/8, $0xe9b5dba5b5c0fbcf
DATA constants<>+0x10(SB)/8, $0x59f111f13956c25b
DATA constants<>+0x18(SB)/8, $0xab1c5ed5923f82a4
DATA constants<>+0x20(SB)/8, $0x12835b01d807aa98
DATA constants<>+0x28(SB)/8, $0x550c7dc3243185be
DATA constants<>+0x30(SB)/8, $0x80deb1fe72be5d74
DATA constants<>+0x38(SB)/8, $0xc19bf1749bdc06a7
DATA constants<>+0x40(SB)/8, $0xefbe4786e49b69c1
DATA constants<>+0x48(SB)/8, $0x240ca1cc0fc19dc6
DATA constants<>+0x50(SB)/8, $0x4a7484aa2de92c6f
DATA constants<>+0x58(SB)/8, $0x76f988da5cb0a9dc
DATA constants<>+0x60(SB)/8, $0xa831c66d983e5152
DATA constants<>+0x68(SB)/8, $0xbf597fc7b00327c8
DATA constants<>+0x70(SB)/8, $0xd5a79147c6e00bf3
DATA constants<>+0x78(SB)/8, $0x1429296706ca6351
DATA constants<>+0x80(SB)/8, $0x2e1b213827b70a85
DATA constants<>+0x88(SB)/8, $0x53380d134d2c6dfc
DATA constants<>+0x90(SB)/8, $0x766a0abb650a7354
DATA constants<>+0x98(SB)/8, $0x92722c8581c2c92e
DATA constants<>+0xa0(SB)/8, $0xa81a664ba2bfe8a1
DATA constants<>+0xa8(SB)/8, $0xc76c51a3c24b8b70
DATA constants<>+0xb0(SB)/8, $0xd6990624d192e819
DATA constants<>+0xb8(SB)/8, $0x106aa070f40e3585
DATA constants<>+0xc0(SB)/8, $0x1e376c0819a4c116
DATA constants<>+0xc8(SB)/8, $0x34b0bcb52748774c
DATA constants<>+0xd0(SB)/8, $0x4ed8aa4a391c0cb3
DATA constants<>+0xd8(SB)/8, $0x682e6ff35b9cca4f
DATA constants<>+0xe0(SB)/8, $0x78a5636f748f82ee
DATA constants<>+0xe8(SB)/8, $0x8cc7020884c87814
DATA constants<>+0xf0(SB)/8, $0xa4506ceb90befffa
DATA constants<>+0xf8(SB)/8, $0xc67178f2bef9a3f7

DATA bflipMask<>+0x00(SB)/8, $0x0405060700010203
DATA bflipMask<>+0x08(SB)/8, $0x0c0d0e0f08090a0b

DATA shuf00BA<>+0x00(SB)/8, $0x0b0a090803020100
DATA shuf00BA<>+0x08(SB)/8, $0xFFFFFFFFFFFFFFFF

DATA shufDC00<>+0x00(SB)/8, $0xFFFFFFFFFFFFFFFF
DATA shufDC00<>+0x08(SB)/8, $0x0b0a090803020100

GLOBL constants<>(SB), 8, $256
GLOBL bflipMask<>(SB), (NOPTR+RODATA), $16
GLOBL shuf00BA<>(SB), (NOPTR+RODATA), $16
GLOBL shufDC00<>(SB), (NOPTR+RODATA), $16
