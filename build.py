#!/usr/bin/env python3
import os
import sys
import argparse


INFO = "\033[1m\033[36m[*]\033[0m "
WARN = "\033[1m\033[31m[!]\033[0m "


def exec_cmd(cmd, ignore_status=False):
    """ Executes a command with some sugar """
    print("\n" + INFO + " %s\n" % cmd)
    status = os.system(cmd)
    if status != 0 and not ignore_status:
        print("\n" + WARN + "Command did not exit cleanly (%s) " % status)
        if input("continue? [Y/n]: ").upper() != 'Y':
            sys.exit(status)
    return status

def docker_rm_volumes():
    exec_cmd("docker volume rm $(docker volume ls -q)", ignore_status=True)

def docker_rm_containers():
    exec_cmd("docker rm $(docker ps -a -q)", ignore_status=True)

def docker_rm_images():
    exec_cmd("docker rmi -f $(docker images -q)", ignore_status=True)

def docker_rm_all():
    docker_rm_containers()
    docker_rm_images()
    docker_rm_volumes()

def build():
    exec_cmd("docker build -t sliver .")

def main(args):
    """ Execute the respective task(s) based on cli args """
    if args.rm_images:
        docker_rm_images()
    if args.rm_containers:
        docker_rm_containers()
    if args.rm_volumes:
        docker_rm_volumes()
    if args.rm_all:
        docker_rm_all()
    if not args.no_build:
        build()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(
        description='automates interactions with docker')

    parser.add_argument('--no-build', '-b',
                        help='do not run build',
                        dest='no_build',
                        action='store_true')

    parser.add_argument('--rm-all', '-rma',
                        help='rm all containers, imagaes, and volumes',
                        dest='rm_all',
                        action='store_true')
    parser.add_argument('--rm-containers', '-rmc',
                        help='rm all containers',
                        dest='rm_containers',
                        action='store_true')
    parser.add_argument('--rm-images', '-rmi',
                        help='rm all images',
                        dest='rm_images',
                        action='store_true')
    parser.add_argument('--rm-volumes', '-rmv',
                        help='rm dangling volumes only',
                        dest='rm_volumes',
                        action='store_true')
    try:
        main(parser.parse_args())
    except KeyboardInterrupt:
        print("\n\n" + WARN + "User stopped process")
