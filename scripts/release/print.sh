# Echos the given string in green
function describe {
    echo -e "\033[0;32m$1\033[0m"
}

# Echos the given string in red
function error {
    echo -e "\033[0;31m$1\033[0m"
}