#!/usr/bin/env bash

# This script is used to fix up the expected output files in the
# oas3_testfiles directory.
# Often when making small changes, a lot of test files are being impacted.
# This script can be used to fix up the expected output files all at once.
#
# 1. make sure the changes have their own tests which succeed, indicating that
#    the failures in the other files are false-positives now.
# 2. commit your changes to ensure you can revert the changes by this script!!
# 3. run the tests with `make test`, to generate all the '*.generated.json' files.
# 4. run this script to copy the generated files over the expected files.
# 5. run the tests again with `make test` to verify that the expected files are
#    now correct.

set -e
# set -x

if [ -d "openapi2kong" ]; then
    pushd openapi2kong
fi
if [ -d "oas3_testfiles" ]; then
    pushd oas3_testfiles
fi

function singlefile() {
    filename="$1"
    postfix="$2"
    destname="${filename%.generated"$postfix".json}.expected$postfix.json"
    echo "$filename  ->  $destname"
    sed 's/\\u003c/</g' "$filename" | sed 's/\\u003e/>/g' > "$destname"
}

# ls -la

for f in ./*.generated.json; do
    singlefile "$f" ""
done
for f in ./*.generated_inso.json; do
    singlefile "$f" "_inso"
done
