#define SHA256_DIGEST_ROW_SIZE 64

#define STATE rdi
#define STATE_P9 DI
#define INP_SIZE rsi
#define INP_SIZE_P9 SI

#define IDX rcx
#define TBL rdx
#define TBL_P9 DX

#define INPUT rax
#define INPUT_P9 AX

#define inp0	r9
#define SCRATCH_P9 R12
#define SCRATCH  r12
#define maskp    r13
#define MASKP_P9 R13
#define mask     r14
#define MASK_P9  R14

#define A       zmm0
#define B       zmm1
#define C       zmm2
#define D       zmm3
#define E       zmm4
#define F       zmm5
#define G       zmm6
#define H       zmm7
#define T1      zmm8
#define TMP0    zmm9
#define TMP1    zmm10
#define TMP2    zmm11
#define TMP3    zmm12
#define TMP4    zmm13
#define TMP5    zmm14
#define TMP6    zmm15

#define W0      zmm16
#define W1      zmm17
#define W2      zmm18
#define W3      zmm19
#define W4      zmm20
#define W5      zmm21
#define W6      zmm22
#define W7      zmm23
#define W8      zmm24
#define W9      zmm25
#define W10     zmm26
#define W11     zmm27
#define W12     zmm28
#define W13     zmm29
#define W14     zmm30
#define W15     zmm31


#define TRANSPOSE16(_r0, _r1, _r2, _r3, _r4, _r5, _r6, _r7, _r8, _r9, _r10, _r11, _r12, _r13, _r14, _r15, _t0, _t1) \
    \
    vshufps _t0, _r0, _r1, 0x44      \
    vshufps _r0, _r0, _r1, 0xEE      \
    vshufps _t1, _r2, _r3, 0x44      \
    vshufps _r2, _r2, _r3, 0xEE      \
                                     \
    vshufps	_r3, _t0, _t1, 0xDD      \
    vshufps	_r1, _r0, _r2, 0x88      \
    vshufps	_r0, _r0, _r2, 0xDD      \
    vshufps	_t0, _t0, _t1, 0x88      \
                                     \
    \
    vshufps _r2, _r4, _r5, 0x44      \
    vshufps _r4, _r4, _r5, 0xEE      \
    vshufps _t1, _r6, _r7, 0x44      \
    vshufps _r6, _r6, _r7, 0xEE      \
                                     \
    vshufps _r7, _r2, _t1, 0xDD      \
    vshufps _r5, _r4, _r6, 0x88      \
    vshufps _r4, _r4, _r6, 0xDD      \
    vshufps _r2, _r2, _t1, 0x88      \
                                     \
    \
    vshufps _r6, _r8, _r9,    0x44   \
    vshufps _r8, _r8, _r9,    0xEE   \
    vshufps _t1, _r10, _r11,  0x44   \
    vshufps _r10, _r10, _r11, 0xEE   \
                                     \
    vshufps _r11, _r6, _t1, 0xDD     \
    vshufps _r9, _r8, _r10, 0x88     \
    vshufps _r8, _r8, _r10, 0xDD     \
    vshufps _r6, _r6, _t1,  0x88     \
                                     \
    \
    vshufps _r10, _r12, _r13, 0x44   \
    vshufps _r12, _r12, _r13, 0xEE   \
    vshufps _t1, _r14, _r15,  0x44   \
    vshufps _r14, _r14, _r15, 0xEE   \
                                     \
    vshufps _r15, _r10, _t1,  0xDD   \
    vshufps _r13, _r12, _r14, 0x88   \
    vshufps _r12, _r12, _r14, 0xDD   \
    vshufps _r10, _r10, _t1,  0x88   \
                                     \
    \
    \
    \
    LEAQ PSHUFFLE_TRANSPOSE16_MASK1<>(SB), BX \
    LEAQ PSHUFFLE_TRANSPOSE16_MASK2<>(SB), R8 \
                                     \
    vmovdqu32 _r14, [rbx]            \
    vpermi2q  _r14, _t0, _r2         \
    vmovdqu32 _t1,  [r8]             \
    vpermi2q  _t1,  _t0, _r2         \
                                     \
    vmovdqu32 _r2, [rbx]             \
    vpermi2q  _r2, _r3, _r7          \
    vmovdqu32 _t0, [r8]              \
    vpermi2q  _t0, _r3, _r7          \
                                     \
    vmovdqu32 _r3, [rbx]             \
    vpermi2q  _r3, _r1, _r5          \
    vmovdqu32 _r7, [r8]              \
    vpermi2q  _r7, _r1, _r5          \
                                     \
    vmovdqu32 _r1, [rbx]             \
    vpermi2q  _r1, _r0, _r4          \
    vmovdqu32 _r5, [r8]              \
    vpermi2q  _r5, _r0, _r4          \
                                     \
    vmovdqu32 _r0, [rbx]             \
    vpermi2q  _r0, _r6, _r10         \
    vmovdqu32 _r4, [r8]              \
    vpermi2q  _r4, _r6, _r10         \
                                     \
    vmovdqu32 _r6, [rbx]             \
    vpermi2q  _r6, _r11, _r15        \
    vmovdqu32 _r10, [r8]             \
    vpermi2q  _r10, _r11, _r15       \
                                     \
    vmovdqu32 _r11, [rbx]            \
    vpermi2q  _r11, _r9, _r13        \
    vmovdqu32 _r15, [r8]             \
    vpermi2q  _r15, _r9, _r13        \
                                     \
    vmovdqu32 _r9, [rbx]             \
    vpermi2q  _r9, _r8, _r12         \
    vmovdqu32 _r13, [r8]             \
    vpermi2q  _r13, _r8, _r12        \
                                     \
    \
    vshuff64x2 _r8, _r14, _r0, 0xEE  \
    vshuff64x2 _r0, _r14, _r0, 0x44  \
                                     \
    vshuff64x2 _r12, _t1, _r4, 0xEE  \
    vshuff64x2 _r4, _t1, _r4, 0x44   \
                                     \
    vshuff64x2 _r14, _r7, _r15, 0xEE \
    vshuff64x2 _t1, _r7, _r15, 0x44  \
                                     \
    vshuff64x2 _r15, _r5, _r13, 0xEE \
    vshuff64x2 _r7, _r5, _r13, 0x44  \
                                     \
    vshuff64x2 _r13, _t0, _r10, 0xEE \
    vshuff64x2 _r5, _t0, _r10, 0x44  \
                                     \
    vshuff64x2 _r10, _r3, _r11, 0xEE \
    vshuff64x2 _t0, _r3, _r11, 0x44  \
                                     \
    vshuff64x2 _r11, _r1, _r9, 0xEE  \
    vshuff64x2 _r3, _r1, _r9, 0x44   \
                                     \
    vshuff64x2 _r9, _r2, _r6, 0xEE   \
    vshuff64x2 _r1, _r2, _r6, 0x44   \
                                     \
    vmovdqu32 _r2, _t0               \
    vmovdqu32 _r6, _t1               \

#define PROCESS_LOOP(_WT, _ROUND, _A, _B, _C, _D, _E, _F, _G, _H)  \
    \
    \
    \
    \
    \
    \
    \
    vpaddd      T1, _H, TMP3           \
    vmovdqu32   TMP0, _E               \
    vprord      TMP1, _E, 6            \
    vprord      TMP2, _E, 11           \
    vprord      TMP3, _E, 25           \
    vpternlogd  TMP0, _F, _G, 0xCA     \
    vpaddd      T1, T1, _WT            \
    vpternlogd  TMP1, TMP2, TMP3, 0x96 \
    vpaddd      T1, T1, TMP0           \
    vpaddd      T1, T1, TMP1           \
    vpaddd      _D, _D, T1             \
                                       \
    vprord      _H, _A, 2              \
    vprord      TMP2, _A, 13           \
    vprord      TMP3, _A, 22           \
    vmovdqu32   TMP0, _A               \
    vpternlogd  TMP0, _B, _C, 0xE8     \
    vpternlogd  _H, TMP2, TMP3, 0x96   \
    vpaddd      _H, _H, TMP0           \
    vpaddd      _H, _H, T1             \
                                       \
    vmovdqu32   TMP3, [TBL + ((_ROUND+1)*64)] \


#define MSG_SCHED_ROUND_16_63(_WT, _WTp1, _WTp9, _WTp14) \
    vprord      TMP4, _WTp14, 17                         \
    vprord      TMP5, _WTp14, 19                         \
    vpsrld      TMP6, _WTp14, 10                         \
    vpternlogd  TMP4, TMP5, TMP6, 0x96                   \
                                                         \
    vpaddd      _WT, _WT, TMP4	                         \
    vpaddd      _WT, _WT, _WTp9	                         \
                                                         \
    vprord      TMP4, _WTp1, 7                           \
    vprord      TMP5, _WTp1, 18                          \
    vpsrld      TMP6, _WTp1, 3                           \
    vpternlogd  TMP4, TMP5, TMP6, 0x96                   \
                                                         \
    vpaddd      _WT, _WT, TMP4	                         \
                                                         \


#define MSG_SCHED_ROUND_00_15(_WT, OFFSET, LABEL)             \
    TESTQ $(1<<OFFSET), MASK_P9                               \
    JE    LABEL                                               \
    MOVQ  OFFSET*24(INPUT_P9), R9                             \
    vmovups _WT, [inp0+IDX]                                   \
LABEL:                                                        \

#define MASKED_LOAD(_WT, OFFSET, LABEL) \
    TESTQ $(1<<OFFSET), MASK_P9         \
    JE    LABEL                         \
    MOVQ  OFFSET*24(INPUT_P9), R9       \
    vmovups _WT,[inp0+IDX]              \
LABEL:                                  \

TEXT ·sha256_x16_avx512(SB), 7, $0
    MOVQ  digests+0(FP), STATE_P9
    MOVQ  scratch+8(FP), SCRATCH_P9
    MOVQ  mask_len+32(FP), INP_SIZE_P9 
    MOVQ  mask+24(FP), MASKP_P9
    MOVQ (MASKP_P9), MASK_P9
    kmovq k1, mask
    LEAQ  inputs+48(FP), INPUT_P9

   
    vmovdqu32 A, [STATE + 0*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 B, [STATE + 1*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 C, [STATE + 2*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 D, [STATE + 3*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 E, [STATE + 4*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 F, [STATE + 5*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 G, [STATE + 6*SHA256_DIGEST_ROW_SIZE]
    vmovdqu32 H, [STATE + 7*SHA256_DIGEST_ROW_SIZE]

    MOVQ  table+16(FP), TBL_P9

    xor IDX, IDX

   
    MASKED_LOAD( W0,  0, skipInput0)
    MASKED_LOAD( W1,  1, skipInput1)
    MASKED_LOAD( W2,  2, skipInput2)
    MASKED_LOAD( W3,  3, skipInput3)
    MASKED_LOAD( W4,  4, skipInput4)
    MASKED_LOAD( W5,  5, skipInput5)
    MASKED_LOAD( W6,  6, skipInput6)
    MASKED_LOAD( W7,  7, skipInput7)
    MASKED_LOAD( W8,  8, skipInput8)
    MASKED_LOAD( W9,  9, skipInput9)
    MASKED_LOAD(W10, 10, skipInput10)
    MASKED_LOAD(W11, 11, skipInput11)
    MASKED_LOAD(W12, 12, skipInput12)
    MASKED_LOAD(W13, 13, skipInput13)
    MASKED_LOAD(W14, 14, skipInput14)
    MASKED_LOAD(W15, 15, skipInput15)

lloop:
    LEAQ PSHUFFLE_BYTE_FLIP_MASK<>(SB), TBL_P9
    vmovdqu32 TMP2, [TBL]

   
    MOVQ  table+16(FP), TBL_P9
    vmovdqu32	TMP3, [TBL]

   
    vmovdqu32 [SCRATCH + 64*0], A
    vmovdqu32 [SCRATCH + 64*1], B
    vmovdqu32 [SCRATCH + 64*2], C
    vmovdqu32 [SCRATCH + 64*3], D
    vmovdqu32 [SCRATCH + 64*4], E
    vmovdqu32 [SCRATCH + 64*5], F
    vmovdqu32 [SCRATCH + 64*6], G
    vmovdqu32 [SCRATCH + 64*7], H

    add IDX, 64

   
    TRANSPOSE16(W0, W1, W2, W3, W4, W5, W6, W7, W8, W9, W10, W11, W12, W13, W14, W15, TMP0, TMP1)

    vpshufb W0, W0, TMP2
    vpshufb W1, W1, TMP2
    vpshufb W2, W2, TMP2
    vpshufb W3, W3, TMP2
    vpshufb W4, W4, TMP2
    vpshufb W5, W5, TMP2
    vpshufb W6, W6, TMP2
    vpshufb W7, W7, TMP2
    vpshufb W8, W8, TMP2
    vpshufb W9, W9, TMP2
    vpshufb W10, W10, TMP2
    vpshufb W11, W11, TMP2
    vpshufb W12, W12, TMP2
    vpshufb W13, W13, TMP2
    vpshufb W14, W14, TMP2
    vpshufb W15, W15, TMP2

   
   
   

    PROCESS_LOOP( W0,  0, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W0,  W1,  W9, W14)
    PROCESS_LOOP( W1,  1, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W1,  W2, W10, W15)
    PROCESS_LOOP( W2,  2, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63( W2,  W3, W11,  W0)
    PROCESS_LOOP( W3,  3, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63( W3,  W4, W12,  W1)
    PROCESS_LOOP( W4,  4, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63( W4,  W5, W13,  W2)
    PROCESS_LOOP( W5,  5, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63( W5,  W6, W14,  W3)
    PROCESS_LOOP( W6,  6, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63( W6,  W7, W15,  W4)
    PROCESS_LOOP( W7,  7, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63( W7,  W8,  W0,  W5)
    PROCESS_LOOP( W8,  8, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W8,  W9,  W1,  W6)
    PROCESS_LOOP( W9,  9, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W9, W10,  W2,  W7)
    PROCESS_LOOP(W10, 10, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63(W10, W11,  W3,  W8)
    PROCESS_LOOP(W11, 11, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63(W11, W12,  W4,  W9)
    PROCESS_LOOP(W12, 12, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63(W12, W13,  W5, W10)
    PROCESS_LOOP(W13, 13, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63(W13, W14,  W6, W11)
    PROCESS_LOOP(W14, 14, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63(W14, W15,  W7, W12)
    PROCESS_LOOP(W15, 15, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63(W15,  W0,  W8, W13)
    PROCESS_LOOP( W0, 16, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W0,  W1,  W9, W14)
    PROCESS_LOOP( W1, 17, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W1,  W2, W10, W15)
    PROCESS_LOOP( W2, 18, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63( W2,  W3, W11,  W0)
    PROCESS_LOOP( W3, 19, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63( W3,  W4, W12,  W1)
    PROCESS_LOOP( W4, 20, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63( W4,  W5, W13,  W2)
    PROCESS_LOOP( W5, 21, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63( W5,  W6, W14,  W3)
    PROCESS_LOOP( W6, 22, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63( W6,  W7, W15,  W4)
    PROCESS_LOOP( W7, 23, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63( W7,  W8,  W0,  W5)
    PROCESS_LOOP( W8, 24, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W8,  W9,  W1,  W6)
    PROCESS_LOOP( W9, 25, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W9, W10,  W2,  W7)
    PROCESS_LOOP(W10, 26, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63(W10, W11,  W3,  W8)
    PROCESS_LOOP(W11, 27, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63(W11, W12,  W4,  W9)
    PROCESS_LOOP(W12, 28, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63(W12, W13,  W5, W10)
    PROCESS_LOOP(W13, 29, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63(W13, W14,  W6, W11)
    PROCESS_LOOP(W14, 30, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63(W14, W15,  W7, W12)
    PROCESS_LOOP(W15, 31, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63(W15,  W0,  W8, W13)
    PROCESS_LOOP( W0, 32, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W0,  W1,  W9, W14)
    PROCESS_LOOP( W1, 33, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W1,  W2, W10, W15)
    PROCESS_LOOP( W2, 34, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63( W2,  W3, W11,  W0)
    PROCESS_LOOP( W3, 35, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63( W3,  W4, W12,  W1)
    PROCESS_LOOP( W4, 36, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63( W4,  W5, W13,  W2)
    PROCESS_LOOP( W5, 37, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63( W5,  W6, W14,  W3)
    PROCESS_LOOP( W6, 38, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63( W6,  W7, W15,  W4)
    PROCESS_LOOP( W7, 39, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63( W7,  W8,  W0,  W5)
    PROCESS_LOOP( W8, 40, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_16_63( W8,  W9,  W1,  W6)
    PROCESS_LOOP( W9, 41, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_16_63( W9, W10,  W2,  W7)
    PROCESS_LOOP(W10, 42, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_16_63(W10, W11,  W3,  W8)
    PROCESS_LOOP(W11, 43, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_16_63(W11, W12,  W4,  W9)
    PROCESS_LOOP(W12, 44, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_16_63(W12, W13,  W5, W10)
    PROCESS_LOOP(W13, 45, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_16_63(W13, W14,  W6, W11)
    PROCESS_LOOP(W14, 46, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_16_63(W14, W15,  W7, W12)
    PROCESS_LOOP(W15, 47, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_16_63(W15,  W0,  W8, W13)

   
    sub INP_SIZE, 1
    JE  lastLoop

   
    ADDQ $8, MASKP_P9
    MOVQ (MASKP_P9), MASK_P9

   
   

    PROCESS_LOOP( W0, 48, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_00_15( W0,  0, skipNext0)
    PROCESS_LOOP( W1, 49, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_00_15( W1,  1, skipNext1)
    PROCESS_LOOP( W2, 50, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_00_15( W2,  2, skipNext2)
    PROCESS_LOOP( W3, 51, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_00_15( W3,  3, skipNext3)
    PROCESS_LOOP( W4, 52, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_00_15( W4,  4, skipNext4)
    PROCESS_LOOP( W5, 53, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_00_15( W5,  5, skipNext5)
    PROCESS_LOOP( W6, 54, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_00_15( W6,  6, skipNext6)
    PROCESS_LOOP( W7, 55, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_00_15( W7,  7, skipNext7)
    PROCESS_LOOP( W8, 56, A, B, C, D, E, F, G, H)
    MSG_SCHED_ROUND_00_15( W8,  8, skipNext8)
    PROCESS_LOOP( W9, 57, H, A, B, C, D, E, F, G)
    MSG_SCHED_ROUND_00_15( W9,  9, skipNext9)
    PROCESS_LOOP(W10, 58, G, H, A, B, C, D, E, F)
    MSG_SCHED_ROUND_00_15(W10, 10, skipNext10)
    PROCESS_LOOP(W11, 59, F, G, H, A, B, C, D, E)
    MSG_SCHED_ROUND_00_15(W11, 11, skipNext11)
    PROCESS_LOOP(W12, 60, E, F, G, H, A, B, C, D)
    MSG_SCHED_ROUND_00_15(W12, 12, skipNext12)
    PROCESS_LOOP(W13, 61, D, E, F, G, H, A, B, C)
    MSG_SCHED_ROUND_00_15(W13, 13, skipNext13)
    PROCESS_LOOP(W14, 62, C, D, E, F, G, H, A, B)
    MSG_SCHED_ROUND_00_15(W14, 14, skipNext14)
    PROCESS_LOOP(W15, 63, B, C, D, E, F, G, H, A)
    MSG_SCHED_ROUND_00_15(W15, 15, skipNext15)

   
    vmovdqu32  TMP2, A
    vmovdqu32 A, [SCRATCH + 64*0]
    vpaddd A{k1}, A, TMP2
    vmovdqu32  TMP2, B
    vmovdqu32 B, [SCRATCH + 64*1]
    vpaddd B{k1}, B, TMP2
    vmovdqu32  TMP2, C
    vmovdqu32 C, [SCRATCH + 64*2]
    vpaddd C{k1}, C, TMP2
    vmovdqu32  TMP2, D
    vmovdqu32 D, [SCRATCH + 64*3]
    vpaddd D{k1}, D, TMP2
    vmovdqu32  TMP2, E
    vmovdqu32 E, [SCRATCH + 64*4]
    vpaddd E{k1}, E, TMP2
    vmovdqu32  TMP2, F
    vmovdqu32 F, [SCRATCH + 64*5]
    vpaddd F{k1}, F, TMP2
    vmovdqu32  TMP2, G
    vmovdqu32 G, [SCRATCH + 64*6]
    vpaddd G{k1}, G, TMP2
    vmovdqu32  TMP2, H
    vmovdqu32 H, [SCRATCH + 64*7]
    vpaddd H{k1}, H, TMP2

    kmovq k1, mask
    JMP lloop

lastLoop:
   
    PROCESS_LOOP( W0, 48, A, B, C, D, E, F, G, H)
    PROCESS_LOOP( W1, 49, H, A, B, C, D, E, F, G)
    PROCESS_LOOP( W2, 50, G, H, A, B, C, D, E, F)
    PROCESS_LOOP( W3, 51, F, G, H, A, B, C, D, E)
    PROCESS_LOOP( W4, 52, E, F, G, H, A, B, C, D)
    PROCESS_LOOP( W5, 53, D, E, F, G, H, A, B, C)
    PROCESS_LOOP( W6, 54, C, D, E, F, G, H, A, B)
    PROCESS_LOOP( W7, 55, B, C, D, E, F, G, H, A)
    PROCESS_LOOP( W8, 56, A, B, C, D, E, F, G, H)
    PROCESS_LOOP( W9, 57, H, A, B, C, D, E, F, G)
    PROCESS_LOOP(W10, 58, G, H, A, B, C, D, E, F)
    PROCESS_LOOP(W11, 59, F, G, H, A, B, C, D, E)
    PROCESS_LOOP(W12, 60, E, F, G, H, A, B, C, D)
    PROCESS_LOOP(W13, 61, D, E, F, G, H, A, B, C)
    PROCESS_LOOP(W14, 62, C, D, E, F, G, H, A, B)
    PROCESS_LOOP(W15, 63, B, C, D, E, F, G, H, A)

   
    vmovdqu32  TMP2, A
    vmovdqu32 A, [SCRATCH + 64*0]
    vpaddd A{k1}, A, TMP2
    vmovdqu32  TMP2, B
    vmovdqu32 B, [SCRATCH + 64*1]
    vpaddd B{k1}, B, TMP2
    vmovdqu32  TMP2, C
    vmovdqu32 C, [SCRATCH + 64*2]
    vpaddd C{k1}, C, TMP2
    vmovdqu32  TMP2, D
    vmovdqu32 D, [SCRATCH + 64*3]
    vpaddd D{k1}, D, TMP2
    vmovdqu32  TMP2, E
    vmovdqu32 E, [SCRATCH + 64*4]
    vpaddd E{k1}, E, TMP2
    vmovdqu32  TMP2, F
    vmovdqu32 F, [SCRATCH + 64*5]
    vpaddd F{k1}, F, TMP2
    vmovdqu32  TMP2, G
    vmovdqu32 G, [SCRATCH + 64*6]
    vpaddd G{k1}, G, TMP2
    vmovdqu32  TMP2, H
    vmovdqu32 H, [SCRATCH + 64*7]
    vpaddd H{k1}, H, TMP2

   
    vmovdqu32 [STATE + 0*SHA256_DIGEST_ROW_SIZE], A
    vmovdqu32 [STATE + 1*SHA256_DIGEST_ROW_SIZE], B
    vmovdqu32 [STATE + 2*SHA256_DIGEST_ROW_SIZE], C
    vmovdqu32 [STATE + 3*SHA256_DIGEST_ROW_SIZE], D
    vmovdqu32 [STATE + 4*SHA256_DIGEST_ROW_SIZE], E
    vmovdqu32 [STATE + 5*SHA256_DIGEST_ROW_SIZE], F
    vmovdqu32 [STATE + 6*SHA256_DIGEST_ROW_SIZE], G
    vmovdqu32 [STATE + 7*SHA256_DIGEST_ROW_SIZE], H

    VZEROUPPER
    RET

DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x000(SB)/8, $0x0405060700010203
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x008(SB)/8, $0x0c0d0e0f08090a0b
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x010(SB)/8, $0x0405060700010203
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x018(SB)/8, $0x0c0d0e0f08090a0b
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x020(SB)/8, $0x0405060700010203
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x028(SB)/8, $0x0c0d0e0f08090a0b
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x030(SB)/8, $0x0405060700010203
DATA PSHUFFLE_BYTE_FLIP_MASK<>+0x038(SB)/8, $0x0c0d0e0f08090a0b
GLOBL PSHUFFLE_BYTE_FLIP_MASK<>(SB), 8, $64

DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x000(SB)/8, $0x0000000000000000
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x008(SB)/8, $0x0000000000000001
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x010(SB)/8, $0x0000000000000008
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x018(SB)/8, $0x0000000000000009
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x020(SB)/8, $0x0000000000000004
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x028(SB)/8, $0x0000000000000005
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x030(SB)/8, $0x000000000000000C
DATA PSHUFFLE_TRANSPOSE16_MASK1<>+0x038(SB)/8, $0x000000000000000D
GLOBL PSHUFFLE_TRANSPOSE16_MASK1<>(SB), 8, $64

DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x000(SB)/8, $0x0000000000000002
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x008(SB)/8, $0x0000000000000003
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x010(SB)/8, $0x000000000000000A
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x018(SB)/8, $0x000000000000000B
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x020(SB)/8, $0x0000000000000006
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x028(SB)/8, $0x0000000000000007
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x030(SB)/8, $0x000000000000000E
DATA PSHUFFLE_TRANSPOSE16_MASK2<>+0x038(SB)/8, $0x000000000000000F
GLOBL PSHUFFLE_TRANSPOSE16_MASK2<>(SB), 8, $64
