// lifted from golang.org/x/sys unix
#include "textflag.h"

TEXT libc_select_trampoline<>(SB), NOSPLIT, $0-0
	JMP libc_select(SB)

GLOBL ·libc_select_trampoline_addr(SB), RODATA, $8
DATA ·libc_select_trampoline_addr(SB)/8, $libc_select_trampoline<>(SB)
