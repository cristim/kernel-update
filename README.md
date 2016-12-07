This is a tiny utility that can ease the process of updating kernels on Ubuntu.

It is automatically installing the latest build from the kernel-ppa/mainline PPA
[repository](http://kernel.ubuntu.com/~kernel-ppa/mainline/).

It supports installing both 'generic' and 'lowlatency' kernel flavors.

# Installation #

    go get github.com/cristim/kernel-update

# Usage #

Assuming `$GOPATH/bin` is in your `PATH`, this should work out of the box:

    kernel-update -flavor lowlatency

