#!/bin/bash

set -e
set -x

eval "$(curl -Ss https://raw.githubusercontent.com/neovim/bot-ci/master/scripts/travis-setup.sh) nightly-x64";

NEOVIM_BIN=$TRAVIS_BUILD_DIR/_neovim/bin/nvim go test
