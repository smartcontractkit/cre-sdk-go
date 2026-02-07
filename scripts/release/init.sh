#!/bin/bash

# Initializes a new release branch from the trunk branch

readonly TRUNK_BRANCH="main"

source ./print.sh
source ./helpers.sh

# accept a version from the user
describe "Initializing a new release branch..."
read -p "Release Version (X.Y.Z): " version

if [[ ! $version =~ ^[0-9]+.[0-9]+.[0-9]+$ ]]
then
    error "Release version must be in semantic versioning format (X.Y.Z)"
    exit 1
fi

# Ensure we don't have any tags which match the provided version
describe "Checking for existing tags..."
if git ls-remote --tags origin | grep -q "v$version"; then
    error "A tag for v$version already exists. Please use a different version number"
    exit 1
fi

current_branch=$(git rev-parse --abbrev-ref HEAD)
release_branch="release/v$version"
remote_trunk_branch="origin/$TRUNK_BRANCH"

# Define the target branch for the release. By default we use the remote trunk, but if the hotfix
# flag is provided, we prompt for a tag to base the release on.
release_target=$remote_trunk_branch
while [[ $# -gt 0 ]]; do
    case "$1" in
        --hotfix)
        describe "Hotfix flag detected. Please provide a tag to base the release on"
        read -p "Tag: " tag
        release_target="tags/v$tag"

        describe "Fetching all remote tags"
        git fetch --all --tags
        ;;
    esac
    shift
done

# Update remote tracking branches
fetch_remote_branches

# Create a release branch from the tip of the remote main branch
describe "Creating $release_branch targeting $release_target..."
git checkout -b $release_branch $release_target
if [ $? -ne 0 ]; then
    error "Failed to create release branch $release_branch"
    exit 1
fi

# Push the release branch to origin
describe "Pushing $release_branch to origin..."
git push -u origin $release_branch
if [ $? -ne 0 ]; then
    error "Failed to publish $release_branch. Rolling back..."

    git checkout $current_branch;
    git branch -D $release_branch;
    exit 1
fi

# Reset the branch back to the user's current branch
describe "Reset local branch back to original branch..."
git checkout $current_branch

describe "Successfully initialized $release_branch!"