# CI and real-database regression

`Build macOS wheels` is the pull-request and main-branch gate. Its first job runs the driver against a fresh DM8 database on a GitHub-hosted ARM Linux runner. The job downloads the image from Dameng's domain, verifies its pinned SHA-256, starts an isolated container, creates a dedicated test account, and runs the P0/P1 tests for pull requests or the complete real-database suite for main and tags. Missing database, failed tests, skipped tests, and an empty JUnit report fail the job.

After that job passes, five macOS ARM jobs build and install wheels for CPython 3.9–3.13. Each job fetches the same official image and extracts DPI headers locally. Headers and image archives are not committed or uploaded as artifacts. The wheel jobs verify import and packaging; the real-database behavior job currently uses CPython 3.10. The standalone `Integration Tests` workflow also runs the full suite nightly and can be started manually.

No GitHub repository database credentials or self-hosted runner are needed. The image is a fixed trial build, so monitor the nightly job and replace the pinned download and checksum if the vendor withdraws it or changes the license behavior. Record the replacement image, server version, and regression result before treating it as a new baseline.

The `DMPYTHON_RELEASE_ENABLED` repository variable must remain unset until the vendored Go driver's public redistribution rights are documented. While unset, CI still builds and verifies wheels but does not upload wheel artifacts or publish a GitHub Release. Publishing to PyPI is a separate step.
