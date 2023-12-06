#!/usr/bin/env python3
"""
Sliver Implant Framework
Copyright (C) 2019  Bishop Fox

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
"""

import os
import sys
import argparse

INFO = "\033[1m\033[36m[*]\033[0m "
WARN = "\033[1m\033[31m[!]\033[0m "
BOLD = "\033[1m"
NORM = "\033[0m"


def exec_cmd(cmd, ignore_status=False):
    """Executes a command with some sugar"""
    print("\n" + INFO + " %s\n" % cmd)
    status = os.system(cmd)
    if status != 0 and not ignore_status:
        print("\n" + WARN + "Command did not exit cleanly (%s) " % status)
        if input("continue? [Y/n]: ").upper() != "Y":
            sys.exit(status)
    return status


def docker_rm_volumes():
    exec_cmd("docker volume rm $(docker volume ls -q)", ignore_status=True)


def docker_rm_containers():
    exec_cmd("docker rm $(docker ps -a -q)", ignore_status=True)


def docker_rm_images():
    exec_cmd("docker rmi -f $(docker images -q)", ignore_status=True)


def docker_prune():
    exec_cmd("docker image prune -f", ignore_status=True)


def docker_rm_all():
    docker_rm_containers()
    docker_rm_images()
    docker_rm_volumes()


def build():
    status = exec_cmd("docker build --target test .")
    if status == 0:
        print("\n\n" + INFO + "Build successful")


def main(args):
    """Execute the respective task(s) based on cli args"""
    if args.rm_images:
        docker_rm_images()
    if args.rm_containers:
        docker_rm_containers()
    if args.rm_volumes:
        docker_rm_volumes()
    if args.rm_all:
        docker_rm_all()

    if not args.no_prune:
        docker_prune()
    if not args.no_build:
        build()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="automates interactions with docker")

    parser.add_argument(
        "--no-build",
        "-b",
        help="do not run build",
        dest="no_build",
        action="store_true",
    )
    parser.add_argument(
        "--no-prune",
        "-np",
        help="do not prune images",
        dest="no_prune",
        action="store_true",
    )

    parser.add_argument(
        "--rm-all",
        "-rma",
        help="rm all containers, images, and volumes",
        dest="rm_all",
        action="store_true",
    )
    parser.add_argument(
        "--rm-containers",
        "-rmc",
        help="rm all containers",
        dest="rm_containers",
        action="store_true",
    )
    parser.add_argument(
        "--rm-images",
        "-rmi",
        help="rm all images",
        dest="rm_images",
        action="store_true",
    )
    parser.add_argument(
        "--rm-volumes",
        "-rmv",
        help="rm dangling volumes only",
        dest="rm_volumes",
        action="store_true",
    )
    try:
        main(parser.parse_args())
    except KeyboardInterrupt:
        print("\n\n" + WARN + "User stopped process")
