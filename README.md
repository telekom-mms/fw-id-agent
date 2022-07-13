# Firewall Identity Agent

Firewall Identity Agent is a systemd user service that logs the current user in
on a Firewall Identity Service. It uses [Trusted Network
Detection](https://github.com/T-Systems-MMS/tnd/) to detect if the host is
currently connected to a trusted network and then logs the user in on the
Firewall Identity Service. Desktop notifications inform the user about the
trusted network and login state.

## Installation

Note: Please see the [install script](/scripts/install.sh) for the individual
install steps and be sure you are OK with the changes this script makes to your
system before you follow these instructions!

You can use the simple [install script](/scripts/install.sh) to install the
firewall identity agent as a systemd user service:

```console
$ ./scripts/install.sh
```

This script installs the example [configuration file](configs/config.json) to
`/etc/fw-id-agent.json` and the example [systemd user
unit](init/fw-id-agent.service). Edit them to match your configuration.

## Usage

If you want to run the Firewall Identity Agent manually, you can run the
executable with the following command line arguments:

```
Usage of fw-id-agent:
  -config file
        Set config file (default "config.json")
  -verbose
        Set verbose output
  -version
        print version
```
