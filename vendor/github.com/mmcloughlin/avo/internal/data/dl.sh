#!/bin/bash -ex

datadir=$(dirname "${BASH_SOURCE[0]}")

dl() {
    url=$1
    name=$(basename ${url})

    mkdir -p ${datadir}
    curl --output ${datadir}/${name} ${url}

    echo "* ${url}"
}

hdr() {
    echo "-----------------------------------------------------------------------------"
    echo $1
    echo "-----------------------------------------------------------------------------"
}

addlicense() {
    repo=$1
    file=$2

    tmp=$(mktemp)
    mv ${file} ${tmp}

    # append to LICENSE file
    {
        hdr "${repo} license"
        echo
        cat ${tmp}
        echo
    } >> ${datadir}/LICENSE

    # include in readme
    echo "### License"
    echo '```'
    cat ${tmp}
    echo '```'

    rm ${tmp}
}

{
    echo '# data'
    echo 'Underlying data files for instruction database.'
    echo

    # golang/arch x86 csv
    repo='golang/arch'
    sha='5a4828bb704534b8a2fa09c791c67d0fb372f472'

    echo "## ${repo}"
    echo 'Files downloaded:'
    echo
    dl https://raw.githubusercontent.com/${repo}/${sha}/x86/x86.v0.2.csv
    dl https://raw.githubusercontent.com/${repo}/${sha}/LICENSE
    addlicense ${repo} ${datadir}/LICENSE

    # opcodes
    repo='Maratyszcza/Opcodes'
    sha='6e2b0cd9f1403ecaf164dea7019dd54db5aea252'

    echo "## ${repo}"
    echo 'Files downloaded:'
    echo
    dl https://raw.githubusercontent.com/${repo}/${sha}/opcodes/x86_64.xml
    dl https://raw.githubusercontent.com/${repo}/${sha}/license.rst
    addlicense ${repo} ${datadir}/license.rst

} > ${datadir}/README.md