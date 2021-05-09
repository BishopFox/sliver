// +build ignore

/*
Input to cgo -godefs.
*/

// +godefs map struct_in_addr [4]byte /* in_addr */
// +godefs map struct_in6_addr [16]byte /* in6_addr */
// +godefs map struct_ [16]byte /* in6_addr */

package netstat

/*
#define __DARWIN_UNIX03 0
#define KERNEL 1
#define XNU_TARGET_OS_OSX 1
#define _DARWIN_USE_64_BIT_INODE
#include <sys/types.h>
#include <sys/uio.h>
#include <sys/un.h>
#include <net/bpf.h>
#include <net/if_dl.h>
#include <net/if_var.h>
#include <net/route.h>
#include <netinet/in.h>

#include <sys/socketvar.h>
#include <netinet/in.h>
#include <netinet/in_pcb.h>
#include <netinet/tcp_var.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#define TCPSTATES
#include <netinet/tcp_fsm.h>
#include <netinet/tcp_seq.h>

enum {
	sizeofPtr = sizeof(void*),
};

union sockaddr_all {
	struct sockaddr s1;	// this one gets used for fields
	struct sockaddr_in s2;	// these pad it out
	struct sockaddr_in6 s3;
	struct sockaddr_un s4;
	struct sockaddr_dl s5;
};

struct sockaddr_any {
	struct sockaddr addr;
	char pad[sizeof(union sockaddr_all) - sizeof(struct sockaddr)];
};

*/
import "C"

// Machine characteristics; for internal use.

const (
	sizeofPtr      = C.sizeofPtr
	sizeofShort    = C.sizeof_short
	sizeofInt      = C.sizeof_int
	sizeofLong     = C.sizeof_long
	sizeofLongLong = C.sizeof_longlong
)

// Basic types

type (
	_C_short     C.short
	_C_int       C.int
	_C_long      C.long
	_C_long_long C.longlong
)

type In6Addr C.struct_in6_addr

type InAddr4in6 C.struct_in_addr_4in6

type XSockbuf C.struct_xsockbuf

type XSocket64 C.struct_xsocket64

type Xinpgen C.struct_xinpgen

type InPCB64ListEntry C.struct_inpcb64_list_entry

type Xinpcb64 C.struct_xinpcb64

type XTCPcb64 C.struct_xtcpcb64
