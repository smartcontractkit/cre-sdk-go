source ./print.sh

# Fetches remote branches from origin
function fetch_remote_branches() {
    # Update remote tracking branches
    describe "Updating remote tracking branches..."
    git fetch origin
}

# Verify that the branch exists
function verify_remote_release_branch() {
    release_branch=$1

    describe "Checking remote origin for release branch..."
    if git ls-remote --exit-code --heads origin $release_branch > /dev/null; then
        echo "Found release branch"
    else
        error "Failed to find release branch"
        exit 1
    fi
}

function ask_version {
    question=$1

    read -p "$question " version

    if [[ ! $version =~ ^[0-9]+.[0-9]+.[0-9]+$ ]]
    then
        echo "Failed to give a proper semver version number, exiting"
        exit 1
    fi

    echo $version
}