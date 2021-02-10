![CI](https://github.com/amplify-edge/booty/workflows/CI/badge.svg)

# (WIP) Booty

### Spec & Design: 

- [Issue 108](https://github.com/amplify-edge/main/issues/108)

## Usage (for devs)

1. Clone this repository
2. Run `make all`
3. Copy the binary in `bin` directory to your `PATH`
4. Run `booty`

## Implemented

1. Download 3rd party binaries and libraries our project needs
2. Install said binaries

## Short Term TODO

1. Also install service, managed by OS' service supervisor, preferably as user service on dev and as system-wide service running under unprivileged user on production.
2. Ability to run each service on the foreground like pingcap's `tiup playground`
3. Ability to self update and check for updates for third party binaries
4. Manage configs and backups for `maintemplatev2`
