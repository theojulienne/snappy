# Devpacks
## Definition
Devpacks are a specific type of Snap packages part of the development
workflow. Devpacks are only available at build time.

* Devpacks are mainly a transport mechanism for software libraries and build
  tools.
* Devpacks typically permit embedding a copy of the provided software
  libraries in an application during build.
* Like apps, devpacks may support one, multiple or all architectures.
* Devpacks are self-contained and do not depend on other devpacks.
* One devpack may provide multiple tools or multiple libraries.
* Devpacks must be coinstallable.

Devpacks are not installed at runtime. Apps must carry all the relevant
runtime bits.

## Usage
### devpack yaml

When creating devpacks, meta/packaging.yaml would contain something like:

    name: foo
    version: 1.2.3
    architecture: host-arch1, host-arch2
    build-architecture: build-arch1, build-arch2
    link-type: shared, static
    type: devpack
    pkg-config-libraries:
      - name: library1
        pkg-config-libdir: library1/lib
    libraries:
      - name: library2
        libdir: library2/lib
        includedir: library2/include
    binaries:
      - name: bin/build-tool
        description: "Description of build tool"

This devpack is to build apps that will run on host-arch1 or host-arch2 (if
unspecified, arch: all is assumed). It provides an additional build tool
described in the `binaries` section that will only work if the build is run
on `build-arch1` or `build-arch2` (by default arch: all is assumed).

This devpack offers two independent libraries; library1 is defined in
pkg-config form; library2 is provided directly.

`link-type: shared, static` indicates that both static and shared libraries
are provided. link-type may also just have the value `shared` or
`static` when only one type of libraries is supported.

The layout of pkg-config-libdir and libdir is in the usual multi-arch triplet
format (e.g. lib/arm-linux-gnueabihf) albeit libdirs themselves also searched.

### Using devpacks

Developers install devpacks into their build environment. Typically a clean
environment is created with snapcraft:
    snapcraft create env1 --base 15.04

This is a container which is started / stopped as needed by snapcraft when
using the environment. By default, this uses a multiarch chroot of the
architecture of the developer system and with support for the default set of
target architectures (e.g. amd64 and armhf) and with some basic development
tools.

Devpacks containing libraries and/or build tools are installed on top with:
    snapcraft addpack env1 foo-devpack

Additional build tools may be installed from Ubuntu with:
    snapcraft run apt-get install make

The build is achieved with:
    snapcraft run make
or whichever the preferred build commands are.

Snapcraft makes some environment variables available to the build process
based on the selected devpacks:

* SNAP_library1_CFLAGS, and LDFLAGS are set by `pkg-config --cflags library1`
  and `pkg-config --libs library1`
* SNAP_library2_CFLAGS and SNAP_library2_LDFLAGS are set to
  `-I/path/to/library1/headers` and `-L/path/to/library2 -llibrary2`
* if a static or shared build is selected, the above variables will be
  adjusted accordingly.

## Open questions

* Are these the right formats (pkg-config and plain lib+headers)?
* Select static vs dynamic via snapcraft --type static/--type shared?
* Build for multiple architectures in one pass?

