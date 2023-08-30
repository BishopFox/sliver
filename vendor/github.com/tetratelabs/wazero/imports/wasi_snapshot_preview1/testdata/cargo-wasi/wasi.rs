use std::env;
use std::fs;
use std::io;
use std::io::{Read,Write};
use std::net::{TcpListener};
use std::os::wasi::io::FromRawFd;
use std::process::exit;
use std::str::from_utf8;

// Until NotADirectory is implemented, read the underlying error raised by
// wasi-libc. See https://github.com/rust-lang/rust/issues/86442
use libc::ENOTDIR;

fn main() {
    let args: Vec<String> = env::args().collect();

    match args[1].as_str() {
        "ls" => {
            main_ls(&args[2]);
            if args.len() == 4 && args[3].as_str() == "repeat" {
                main_ls(&args[2]);
            }
        }
        "stat" => main_stat(),
        "sock" => main_sock(),
        _ => {
            writeln!(io::stderr(), "unknown command: {}", args[1]).unwrap();
            exit(1);
        }
    }
}

fn main_ls(dir_name: &String) {
    match fs::read_dir(dir_name) {
        Ok(paths) => {
            for ent in paths.into_iter() {
                println!("{}", ent.unwrap().path().display());
            }
        }
        Err(e) => {
            if let Some(error_code) = e.raw_os_error() {
                if error_code == ENOTDIR {
                    println!("ENOTDIR");
                } else {
                    println!("errno=={}", error_code);
                }
            } else {
                writeln!(io::stderr(), "failed to read directory: {}", e).unwrap();
            }
        }
    }
}

extern crate libc;

fn main_stat() {
    unsafe {
        println!("stdin isatty: {}", libc::isatty(0) != 0);
        println!("stdout isatty: {}", libc::isatty(1) != 0);
        println!("stderr isatty: {}", libc::isatty(2) != 0);
        println!("/ isatty: {}", libc::isatty(3) != 0);
    }
}

fn main_sock() {
    // Get a listener from the pre-opened file descriptor.
    // The listener is the first pre-open, with a file-descriptor of 3.
    let listener = unsafe { TcpListener::from_raw_fd(3) };
    for conn in listener.incoming() {
        match conn {
            Ok(mut conn) => {
                // Do a blocking read of up to 32 bytes.
                // Note: the test should write: "wazero", so that's all we should read.
                let mut data = [0 as u8; 32];
                match conn.read(&mut data) {
                    Ok(size) => {
                        let text = from_utf8(&data[0..size]).unwrap();
                        println!("{}", text);

                        // Exit instead of accepting another connection.
                        exit(0);
                    },
                    Err(e) => writeln!(io::stderr(), "failed to read data: {}", e).unwrap(),
                } {}
            }
            Err(e) => writeln!(io::stderr(), "failed to read connection: {}", e).unwrap(),
        }
    }
}
