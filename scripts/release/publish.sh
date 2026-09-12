#!/bin/bash

# You'll need to update these to point to your actual print and helpers scripts
source ./print.sh
source ./helpers.sh

describe "Generating and pushing release tags"
read -p "Release Version (X.Y.Z): " version

if [[ ! $version =~ ^[0-9]+.[0-9]+.[0-9]+$ ]]
then
    error "Release version must be in semantic versioning format (X.Y.Z)"
    exit 1
fi

release_branch="release/v$version"
base_release_tag="v$version"

# --- Prompt for Annotated Tag Message ---
read -p "Enter Tag Message (e.g., v${version} release): " TAG_MESSAGE
if [ -z "$TAG_MESSAGE" ]; then
    error "Tag message cannot be empty. Aborting."
    exit 1
fi

# Verify that the release branch exists
describe "Verifying existence of remote branch: $release_branch"
if ! git ls-remote --exit-code origin $release_branch > /dev/null 2>&1
then
    error "Remote branch '$release_branch' does not exist on origin."
    exit 1
fi

# Ask for the stage of release
read -p "Version Stage [beta/stable]: " versionStage

if [[ ! $versionStage =~ ^beta$|^stable$ ]]
then
    echo "Type must be either beta or stable."
    exit 1
fi

# Determine the final tag(s) and stage
final_tags=()
pre_release_suffix=""

if [ $versionStage = "stable" ]
then
    read -p "Deploy a stable production tag? ([N|n]o/[Y|y]es): " reply
    if [[ $reply =~ ^[Yy]$|^[Yy]es$ ]]
    then
        echo "Preparing stable tag: $base_release_tag"
        final_tags+=("$base_release_tag")
    elif [[ $reply =~ ^[Nn]$|^[Nn]o$ ]]
    then
        echo "Aborting tag publication."
        exit 0
    else
        echo "Expected Yes/yes/Y/y or No/no/N/n"
        exit 1
    fi
fi


if [ $versionStage = "beta" ]
then
    # Fetch the list of remote tags matching the version to determine the next iteration counter
    describe "Determining beta iteration using git ls-remote..."
    
    # Fetch all existing tags for the current version (vX.Y.Z*)
    remote_tag_refs=$(git ls-remote --tags origin "$base_release_tag*" | awk '{print $2}')

    iter=-1
    # Read the tags line by line
    while IFS= read -r ref; do
        tag_name=${ref##refs/tags/}
        
        if [[ $tag_name == *"-beta."* ]]
        then
            number=${tag_name##*.}
            if [[ $number =~ ^[0-9]+$ ]]
            then
                if [ $number -gt $iter ]
                then
                    iter=$number
                fi
            fi
        fi
    done <<< "$remote_tag_refs"

    nextiter=$((iter+1))
    
    # If no previous tags were found, start at beta.0
    if [ $iter -eq -1 ]; then
        nextiter=0
    fi

    read -p "Use $nextiter as the next beta iteration value? ([N|n]o/[Y|y]es): " reply
    if [[ $reply =~ ^[Yy]$|^[Yy]es$ ]]
    then
        echo "Using $nextiter as the next iteration value."
    elif [[ $reply =~ ^[Nn]$|^[Nn]o$ ]]
    then
        read -p "Enter a custom iteration: " customiter
        if [[ ! $customiter =~ ^[0-9]+$ ]]; then
            error "Custom iteration must be a positive integer."
            exit 1
        fi
        nextiter=$customiter
    else
        echo "Expected Yes/yes/Y/y or No/no/N/n"
        exit 1
    fi
    
    # Set the FINAL suffix
    pre_release_suffix="-beta.$nextiter"
    
    # 1. Add the BASE tag (e.g., v1.0.0-beta.0)
    final_tags+=("$base_release_tag$pre_release_suffix") 
    
    # Component prefixes for additional tags
    component_prefixes=(
        "generator/protoc-gen-cre/"
        "capabilities/scheduler/cron/"
        "capabilities/networking/http/"
        "capabilities/blockchain/evm/"
    )
    
    # 2. Add the COMPONENT tags (e.g., generator/protoc-gen-cre/v1.0.0-beta.0)
    for prefix in "${component_prefixes[@]}"; do
        final_tags+=("$prefix$base_release_tag$pre_release_suffix")
    done
fi

describe "Creating and pushing tags targeting $release_branch"

# Ensure we are on the correct branch to create the tag
git fetch origin $release_branch > /dev/null 2>&1
TARGET_COMMIT=$(git rev-parse "origin/$release_branch")
if [ -z "$TARGET_COMMIT" ]; then
    error "Could not find commit for remote branch origin/$release_branch."
    exit 1
fi

for tag in "${final_tags[@]}"; do
    describe "Processing tag: $tag"
    
    # Check if tag already exists locally or remotely
    if git rev-parse --quiet --verify "refs/tags/$tag" > /dev/null || git ls-remote --tags origin "$tag" | grep -q "$tag"
    then
        echo "Tag $tag already exists locally or remotely. Skipping."
        continue
    fi
    
    # Create the ANNOTATED tag on the target commit
    echo "Creating annotated tag $tag on commit $TARGET_COMMIT with message: \"$TAG_MESSAGE\""
    if ! git tag -a "$tag" -m "$TAG_MESSAGE" "$TARGET_COMMIT"
    then
        error "Failed to create local tag $tag."
        exit 1
    fi

    # Push the tag to the remote
    echo "Pushing tag $tag to origin"
    if ! git push origin "$tag"
    then
        error "Failed to push tag $tag to origin. Rolling back local tag..."
        git tag -d "$tag" # Delete local tag on failure
        exit 1
    fi
    echo "Successfully pushed $tag."
done

echo "Successfully generated and pushed all required tags!"
exit 0