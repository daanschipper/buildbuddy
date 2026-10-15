#!/usr/bin/env bash
set -e

# Prints the latest BuildBuddy version tag, like "v2.12.8"
# Only tags contained in HEAD: a release branch may carry newer unrelated upstream tags.
git tag -l 'v*' --sort=creatordate --merged HEAD |
    perl -nle 'if (/^v\d+\.\d+\.\d+$/) { print $_ }' |
    tail -n1

