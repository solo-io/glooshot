#!/bin/bash

set -ex

protoc --version

if [ ! -f .gitignore ]; then
  echo "_output" > .gitignore
fi

git init
git add .
git commit -m "set up dummy repo for diffing" -q --allow-empty

git clone https://github.com/solo-io/solo-kit /workspace/gopath/src/github.com/solo-io/solo-kit
git clone https://github.com/solo-io/gloo /workspace/gopath/src/github.com/solo-io/gloo

make update-deps
make pin-repos

PATH=/workspace/gopath/bin:$PATH

set +e

# write output to a random* file, print only if there's an error
# *mktemp guarantees that the file will be available
GEN_OUTPUT_FILE=`mktemp`
make generated-code -B >> $GEN_OUTPUT_FILE
if [[ $? -ne 0 ]]; then
  echo "Code generation failed"
  echo "output from generation:"
  cat $GEN_OUTPUT_FILE
  exit 1;
fi
if [[ $(git status --porcelain | wc -l) -ne 0 ]]; then
  echo "Generating code produced a non-empty diff"
  echo "Try running 'make update-deps generated-code -B' then re-pushing."
  git status --porcelain
  git diff | cat
  exit 1;
fi
