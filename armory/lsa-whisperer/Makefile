# lsa-whisperer BOF - Build for Sliver armory
#
# Builds three BOF modules targeting Windows x64 and x86:
#   msv1_0_bof  - DPAPI credential key extraction, NTLMv1 response generation
#   kerberos_bof - Kerberos ticket enumeration, dump, and purge
#   cloudap_bof  - Azure/Entra ID SSO token extraction
#
# Requirements: mingw-w64 cross-compiler
#   apt install gcc-mingw-w64-x86-64 gcc-mingw-w64-i686

CC_X64 = x86_64-w64-mingw32-gcc
CC_X86 = i686-w64-mingw32-gcc

CFLAGS = -Os -DBOF -c
INCLUDES = -I include

BUILD_DIR = build
SRC_COMMON = src/common/lsa_common.c

.PHONY: all clean msv1_0 kerberos cloudap package

all: msv1_0 kerberos cloudap
	@echo "[+] All BOF modules built successfully"
	@ls -la $(BUILD_DIR)/*.o

# ── MSV1_0 Module (credkey, strongcredkey, ntlmv1) ──────────
msv1_0: $(BUILD_DIR)/msv1_0_bof.x64.o $(BUILD_DIR)/msv1_0_bof.x86.o

$(BUILD_DIR)/msv1_0_bof.x64.o: src/msv1_0/msv1_0_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X64) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/msv1_0/msv1_0_bof.c \
		-o $@

$(BUILD_DIR)/msv1_0_bof.x86.o: src/msv1_0/msv1_0_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X86) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/msv1_0/msv1_0_bof.c \
		-o $@

# ── Kerberos Module (klist, dump, purge) ─────────────────────
kerberos: $(BUILD_DIR)/kerberos_bof.x64.o $(BUILD_DIR)/kerberos_bof.x86.o

$(BUILD_DIR)/kerberos_bof.x64.o: src/kerberos/kerberos_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X64) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/kerberos/kerberos_bof.c \
		-o $@

$(BUILD_DIR)/kerberos_bof.x86.o: src/kerberos/kerberos_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X86) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/kerberos/kerberos_bof.c \
		-o $@

# ── CloudAP Module (ssocookie, devicessotoken, enterprisesso, info)
cloudap: $(BUILD_DIR)/cloudap_bof.x64.o $(BUILD_DIR)/cloudap_bof.x86.o

$(BUILD_DIR)/cloudap_bof.x64.o: src/cloudap/cloudap_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X64) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/cloudap/cloudap_bof.c \
		-o $@

$(BUILD_DIR)/cloudap_bof.x86.o: src/cloudap/cloudap_bof.c $(SRC_COMMON) include/*.h
	@mkdir -p $(BUILD_DIR)
	$(CC_X86) $(CFLAGS) $(INCLUDES) \
		-include $(SRC_COMMON) \
		src/cloudap/cloudap_bof.c \
		-o $@

# ── Package for armory (tar.gz with extension.json + .o files) ──
package: all
	@echo "[+] Packaging for Sliver armory..."
	@mkdir -p $(BUILD_DIR)/pkg/lsa-whisperer
	@cp extension.json $(BUILD_DIR)/pkg/lsa-whisperer/
	@cp $(BUILD_DIR)/*.o $(BUILD_DIR)/pkg/lsa-whisperer/
	@cd $(BUILD_DIR)/pkg && tar czf ../lsa-whisperer.tar.gz lsa-whisperer/
	@echo "[+] Package created: $(BUILD_DIR)/lsa-whisperer.tar.gz"
	@echo "[+] Install with: extensions load $(BUILD_DIR)/pkg/lsa-whisperer"

clean:
	rm -rf $(BUILD_DIR)
