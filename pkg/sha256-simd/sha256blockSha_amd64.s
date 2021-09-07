//+build !noasm,!appengine

#include "textflag.h"

DATA K<>+0x00(SB)/4, $0x428a2f98
DATA K<>+0x04(SB)/4, $0x71374491
DATA K<>+0x08(SB)/4, $0xb5c0fbcf
DATA K<>+0x0c(SB)/4, $0xe9b5dba5
DATA K<>+0x10(SB)/4, $0x3956c25b
DATA K<>+0x14(SB)/4, $0x59f111f1
DATA K<>+0x18(SB)/4, $0x923f82a4
DATA K<>+0x1c(SB)/4, $0xab1c5ed5
DATA K<>+0x20(SB)/4, $0xd807aa98
DATA K<>+0x24(SB)/4, $0x12835b01
DATA K<>+0x28(SB)/4, $0x243185be
DATA K<>+0x2c(SB)/4, $0x550c7dc3
DATA K<>+0x30(SB)/4, $0x72be5d74
DATA K<>+0x34(SB)/4, $0x80deb1fe
DATA K<>+0x38(SB)/4, $0x9bdc06a7
DATA K<>+0x3c(SB)/4, $0xc19bf174
DATA K<>+0x40(SB)/4, $0xe49b69c1
DATA K<>+0x44(SB)/4, $0xefbe4786
DATA K<>+0x48(SB)/4, $0x0fc19dc6
DATA K<>+0x4c(SB)/4, $0x240ca1cc
DATA K<>+0x50(SB)/4, $0x2de92c6f
DATA K<>+0x54(SB)/4, $0x4a7484aa
DATA K<>+0x58(SB)/4, $0x5cb0a9dc
DATA K<>+0x5c(SB)/4, $0x76f988da
DATA K<>+0x60(SB)/4, $0x983e5152
DATA K<>+0x64(SB)/4, $0xa831c66d
DATA K<>+0x68(SB)/4, $0xb00327c8
DATA K<>+0x6c(SB)/4, $0xbf597fc7
DATA K<>+0x70(SB)/4, $0xc6e00bf3
DATA K<>+0x74(SB)/4, $0xd5a79147
DATA K<>+0x78(SB)/4, $0x06ca6351
DATA K<>+0x7c(SB)/4, $0x14292967
DATA K<>+0x80(SB)/4, $0x27b70a85
DATA K<>+0x84(SB)/4, $0x2e1b2138
DATA K<>+0x88(SB)/4, $0x4d2c6dfc
DATA K<>+0x8c(SB)/4, $0x53380d13
DATA K<>+0x90(SB)/4, $0x650a7354
DATA K<>+0x94(SB)/4, $0x766a0abb
DATA K<>+0x98(SB)/4, $0x81c2c92e
DATA K<>+0x9c(SB)/4, $0x92722c85
DATA K<>+0xa0(SB)/4, $0xa2bfe8a1
DATA K<>+0xa4(SB)/4, $0xa81a664b
DATA K<>+0xa8(SB)/4, $0xc24b8b70
DATA K<>+0xac(SB)/4, $0xc76c51a3
DATA K<>+0xb0(SB)/4, $0xd192e819
DATA K<>+0xb4(SB)/4, $0xd6990624
DATA K<>+0xb8(SB)/4, $0xf40e3585
DATA K<>+0xbc(SB)/4, $0x106aa070
DATA K<>+0xc0(SB)/4, $0x19a4c116
DATA K<>+0xc4(SB)/4, $0x1e376c08
DATA K<>+0xc8(SB)/4, $0x2748774c
DATA K<>+0xcc(SB)/4, $0x34b0bcb5
DATA K<>+0xd0(SB)/4, $0x391c0cb3
DATA K<>+0xd4(SB)/4, $0x4ed8aa4a
DATA K<>+0xd8(SB)/4, $0x5b9cca4f
DATA K<>+0xdc(SB)/4, $0x682e6ff3
DATA K<>+0xe0(SB)/4, $0x748f82ee
DATA K<>+0xe4(SB)/4, $0x78a5636f
DATA K<>+0xe8(SB)/4, $0x84c87814
DATA K<>+0xec(SB)/4, $0x8cc70208
DATA K<>+0xf0(SB)/4, $0x90befffa
DATA K<>+0xf4(SB)/4, $0xa4506ceb
DATA K<>+0xf8(SB)/4, $0xbef9a3f7
DATA K<>+0xfc(SB)/4, $0xc67178f2
GLOBL K<>(SB), RODATA|NOPTR, $256

DATA SHUF_MASK<>+0x00(SB)/8, $0x0405060700010203
DATA SHUF_MASK<>+0x08(SB)/8, $0x0c0d0e0f08090a0b
GLOBL SHUF_MASK<>(SB), RODATA|NOPTR, $16

TEXT ·blockSha(SB), NOSPLIT, $0-32
	MOVQ      h+0(FP), DX
	MOVQ      message_base+8(FP), SI
	MOVQ      message_len+16(FP), DI
	LEAQ      -64(SI)(DI*1), DI
	MOVOU     (DX), X2
	MOVOU     16(DX), X1
	MOVO      X2, X3
	PUNPCKLLQ X1, X2
	PUNPCKHLQ X1, X3
	PSHUFD    $0x27, X2, X2
	PSHUFD    $0x27, X3, X3
	MOVO      SHUF_MASK<>(SB), X15
	LEAQ      K<>(SB), BX

	JMP TEST

LOOP:
	MOVO X2, X12
	MOVO X3, X13


	MOVOU  (SI), X4
	MOVOU  16(SI), X5
	MOVOU  32(SI), X6
	MOVOU  48(SI), X7
	PSHUFB X15, X4
	PSHUFB X15, X5
	PSHUFB X15, X6
	PSHUFB X15, X7

#define ROUND456 \
	PADDL  X5, X0                    \
	LONG   $0xdacb380f               \
	MOVO   X5, X1                    \
	LONG   $0x0f3a0f66; WORD $0x04cc \
	PADDL  X1, X6                    \
	LONG   $0xf5cd380f               \
	PSHUFD $0x4e, X0, X0             \
	LONG   $0xd3cb380f               \
	LONG   $0xe5cc380f              

#define ROUND567 \
	PADDL  X6, X0                    \
	LONG   $0xdacb380f               \
	MOVO   X6, X1                    \
	LONG   $0x0f3a0f66; WORD $0x04cd \
	PADDL  X1, X7                    \
	LONG   $0xfecd380f               \
	PSHUFD $0x4e, X0, X0             \
	LONG   $0xd3cb380f               \
	LONG   $0xeecc380f              

#define ROUND674 \
	PADDL  X7, X0                    \
	LONG   $0xdacb380f               \
	MOVO   X7, X1                    \
	LONG   $0x0f3a0f66; WORD $0x04ce \
	PADDL  X1, X4                    \
	LONG   $0xe7cd380f               \
	PSHUFD $0x4e, X0, X0             \
	LONG   $0xd3cb380f               \
	LONG   $0xf7cc380f              

#define ROUND745 \
	PADDL  X4, X0                    \
	LONG   $0xdacb380f               \
	MOVO   X4, X1                    \
	LONG   $0x0f3a0f66; WORD $0x04cf \
	PADDL  X1, X5                    \
	LONG   $0xeccd380f               \
	PSHUFD $0x4e, X0, X0             \
	LONG   $0xd3cb380f               \
	LONG   $0xfccc380f              


	MOVO   (BX), X0
	PADDL  X4, X0
	LONG   $0xdacb380f  
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f  


	MOVO   1*16(BX), X0
	PADDL  X5, X0
	LONG   $0xdacb380f  
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f  
	LONG   $0xe5cc380f  


	MOVO   2*16(BX), X0
	PADDL  X6, X0
	LONG   $0xdacb380f  
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f  
	LONG   $0xeecc380f  

	MOVO 3*16(BX), X0; ROUND674 
	MOVO 4*16(BX), X0; ROUND745 
	MOVO 5*16(BX), X0; ROUND456 
	MOVO 6*16(BX), X0; ROUND567 
	MOVO 7*16(BX), X0; ROUND674 
	MOVO 8*16(BX), X0; ROUND745 
	MOVO 9*16(BX), X0; ROUND456 
	MOVO 10*16(BX), X0; ROUND567
	MOVO 11*16(BX), X0; ROUND674
	MOVO 12*16(BX), X0; ROUND745


	MOVO   13*16(BX), X0
	PADDL  X5, X0
	LONG   $0xdacb380f              
	MOVO   X5, X1
	LONG   $0x0f3a0f66; WORD $0x04cc
	PADDL  X1, X6
	LONG   $0xf5cd380f              
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f              


	MOVO   14*16(BX), X0
	PADDL  X6, X0
	LONG   $0xdacb380f              
	MOVO   X6, X1
	LONG   $0x0f3a0f66; WORD $0x04cd
	PADDL  X1, X7
	LONG   $0xfecd380f              
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f              


	MOVO   15*16(BX), X0
	PADDL  X7, X0
	LONG   $0xdacb380f  
	PSHUFD $0x4e, X0, X0
	LONG   $0xd3cb380f  

	PADDL X12, X2
	PADDL X13, X3

	ADDQ $64, SI

TEST:
	CMPQ SI, DI
	JBE  LOOP

	PSHUFD $0x4e, X3, X0
	LONG   $0x0e3a0f66; WORD $0xf0c2
	PSHUFD $0x4e, X2, X1
	LONG   $0x0e3a0f66; WORD $0x0fcb
	PSHUFD $0x1b, X0, X0
	PSHUFD $0x1b, X1, X1

	MOVOU X0, (DX)
	MOVOU X1, 16(DX)

	RET
