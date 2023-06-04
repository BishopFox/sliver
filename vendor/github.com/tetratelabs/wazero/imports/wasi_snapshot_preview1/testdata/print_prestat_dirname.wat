;; $$print_prestat_dirname is a WASI command which copies the first preopen dirname to stdout.
(module $print_prestat_dirname
    (import "wasi_snapshot_preview1" "fd_prestat_get"
        (func $wasi.fd_prestat_get (param $fd i32) (param $result.prestat i32) (result (;errno;) i32)))

    (import "wasi_snapshot_preview1" "fd_prestat_dir_name"
        (func $wasi.fd_prestat_dir_name (param $fd i32) (param $result.path i32)  (param $result.path_len i32) (result (;errno;) i32)))

    (import "wasi_snapshot_preview1" "fd_write"
        (func $wasi.fd_write (param $fd i32) (param $iovs i32) (param $iovs_len i32) (param $result.size i32) (result (;errno;) i32)))

    (memory (export "memory") 1 1)

    (func $main (export "_start")
        ;; First, we need to know the size of the prestat dir name.
        (call $wasi.fd_prestat_get
            (i32.const 3) ;; preopen FD
            (i32.const 0) ;; where to write prestat
        )
        drop ;; ignore the errno returned

        ;; Next, write the dir name to offset 8 (past the prestat).
        (call $wasi.fd_prestat_dir_name
            (i32.const 3) ;; preopen FD
            (i32.const 8) ;; where to write dir_name
            (i32.load (i32.const 4)) ;; length is the last part of the prestat
        )
        drop ;; ignore the errno returned

        ;; Now, convert the prestat to an iovec [offset, len] writing offset=8.
        (i32.store (i32.const 0) (i32.const 8))

        ;; Finally, write the dirname to stdout via its iovec [offset, len].
        (call $wasi.fd_write
            (i32.const 1) ;; stdout
            (i32.const 0) ;; where's the iovec
            (i32.const 1) ;; only one iovec
            (i32.const 0) ;; overwrite the iovec with the ignored result.
        )
        drop ;; ignore the errno returned
    )
)
