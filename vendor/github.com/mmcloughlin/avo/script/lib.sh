# Print an error message to the terminal.
print_error() {
    local red='\033[0;31m'
    local nocolor='\033[0m'
    printf "${red}error${nocolor}\t$1\n"
}
