#!/bin/bash

source ./print.sh

describe "Fixing release branch"
read -p "Release Version (X.Y.Z): " version

if [[ ! $version =~ ^[0-9]+.[0-9]+.[0-9]+$ ]]
then
    error "Release version must be in semantic versioning format (X.Y.Z)"
    exit 1
fi

# Get the cherry-pick commit hash
read -p "Enter the commit hash to cherry pick: " commitHash

git show $commitHash

current_branch=$(git rev-parse --abbrev-ref HEAD)
release_branch="release/v$version"
fix_branch="fix/v$version/$commitHash"

# Check that the branch exists
describe "Checking remote origin for release branch..."
if git ls-remote --exit-code --heads origin $release_branch > /dev/null; then
    echo "Found release branch"
else
    error "Failed to find release branch"
    exit 1
fi

# Checkout the release branch
describe "Updating remote tracking branches..."
git fetch origin

# Create a new branch from the release branch
describe "Creating fix branch..."
git checkout -B $fix_branch origin/$release_branch

# Cherry pick the commit
describe "Cherry picking commit onto fix branch..."
if git cherry-pick -S $commitHash > /dev/null; then
    echo "Cherry pick successful"
else
    error "Failed to cherry pick commit. Rolling back..."

    git cherry-pick --abort
    git checkout $current_branch
    git branch -D $fix_branch

    exit 1
fi

# Push the fix branch
describe "Pushing fix branch to origin..."
git push origin $fix_branch

# Create a pull request
describe "Creating pull request to merge fix branch into release branch..."
gh pr create --fill --base $release_branch --head $fix_branch --label "release-fix"

# Reset the branch back to the user's current branch
describe "Reset local branch back to original branch..."
git checkout $current_branch