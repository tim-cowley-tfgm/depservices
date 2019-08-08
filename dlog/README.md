# dlog

This is a simple wrapper around the standard Go log package.

It adds debug functionality, which can be enabled or disabled with the use of
build flags; e.g:

`go build -tags=debug` will enable the debug output

If the build tag `debug` is not present, there will be no debug output.
