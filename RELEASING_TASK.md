# Releasing Task

The release process of Task is done is done with the help of
[GoReleaser][goreleaser]. You can test the release process locally by calling
the `test-release` task of the Taskfile.

The Travis CI should release automatically when a new
Git tag is pushed to master, either for the artifact uploading (raw executables
and DEB and RPM packages) and publishing of a new version in the
[Homebrew tap][homebrewtap].

# Snapcraft

The exception is the publishing of a new version of the
[snap package][snappackage]. This current require two steps after publishing
the binaries:

* Updating the current version on [snapcraft.yaml][snapcraftyaml];
* Moving either the `i386` and `amd64` new artifacts to the stable channel on
the [Snapscraft dashboard][snapcraftdashboard]

[goreleaser]: https://goreleaser.com/#continuous_integration
[homebrewtap]: https://github.com/go-task/homebrew-tap
[snappackage]: https://github.com/go-task/snap
[snapcraftyaml]: https://github.com/go-task/snap/blob/master/snap/snapcraft.yaml#L2
[snapcraftdashboard]: https://dashboard.snapcraft.io/
