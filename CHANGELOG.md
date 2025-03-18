# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [unreleased]
### Fixed
- 429 status code responses by adding a random timer to the pihole api.

## [v0.2.1]
### Fixed
- Cloudflare access in client login.

## [v0.2.0]
### Fixed
- Blocking now uses the `/api/dns/blocking` endpoint.
- DNS Recods now uses the `/api/config/dns/hosts` endpoint.
- Groups now uses the `/api/groups` endpoint.

## [v0.1.2]
### Fixed
- Login now uses json request.

## [v0.1.1]
### Fixed
- Login now uses the `/api/auth` endpoint.

## [v0.1.0]
- Forked from [ryanwholey/terraform-provider-pihole]

### Added
- Cloudflare access support.

[unreleased]: https://github.com/iolave/terraform-provider-pihole/compare/v0.2.1...master
[v0.2.1]: https://github.com/iolave/terraform-provider-pihole/releases/tag/v0.2.1
[v0.2.0]: https://github.com/iolave/terraform-provider-pihole/releases/tag/v0.2.0
[v0.1.2]: https://github.com/iolave/terraform-provider-pihole/releases/tag/v0.1.2
[v0.1.1]: https://github.com/iolave/terraform-provider-pihole/releases/tag/v0.1.1
[v0.1.0]: https://github.com/iolave/terraform-provider-pihole/releases/tag/v0.1.0
[ryanwholey/terraform-provider-pihole]: https://github.com/ryanwholey/terraform-provider-pihole
